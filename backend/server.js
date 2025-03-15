import * as fs from 'fs'
import Fastify from 'fastify'
import fastifyCors from '@fastify/cors'
import { ChatOllama } from "@langchain/ollama";
import { OllamaEmbeddings } from '@langchain/ollama';
import { Client } from '@elastic/elasticsearch';
import { console } from 'inspector';

const client = new Client({
  nodes: [process.env.ELASTICSEARCH_HOSTS], // Array of Elasticsearch node addresses
  auth: {
    username: process.env.ELASTICSEARCH_USERNAME,
    password: process.env.ELASTICSEARCH_PASSWORD
  }
})

const resp = await client.info();

console.log("🔎🔎🔎", resp);

// Create the Ollama Client
const llm = new ChatOllama({
  model: process.env.LLM_CHAT || "qwen2.5:3b",
  baseUrl: process.env.OLLAMA_BASE_URL || "http://ollama:11434",
  temperature: parseFloat(process.env.OPTION_TEMPERATURE) || 0.0,
  repeatLastN: parseInt(process.env.OPTION_REPEAT_LAST_N) || 2,
  repeatPenalty: parseFloat(process.env.OPTION_REPEAT_PENALTY) || 2.2,
  topK: parseInt(process.env.OPTION_TOP_K) || 10,
  topP: parseFloat(process.env.OPTION_TOP_P) || 0.5,
  numCtx: parseInt(process.env.OPTION_NUM_CTX) || 2048,
})

const llmEmbeddings = new OllamaEmbeddings({
  model: process.env.LLM_EMBEDDINGS || "mxbai-embed-large",
  baseUrl: process.env.OLLAMA_BASE_URL || "http://ollama:11434",
})

let directoryTreePath = process.env.DIRECTORY_TREE_PATH || "../data/tree.txt"
let directoryTree = fs.readFileSync(directoryTreePath, 'utf8')

// Load the system instructions
let systemInstructions = fs.readFileSync(process.env.SYSTEM_INSTRUCTIONS_PATH, 'utf8')

const fastify = Fastify({ logger: true })

// Initialize a Map to store conversations by session
const conversationMemory = new Map()

// Helper function to get or create a conversation history
function getConversationHistory(sessionId, maxTurns = parseInt(process.env.HISTORY_MESSAGES)) {
  if (!conversationMemory.has(sessionId)) {
    conversationMemory.set(sessionId, [])
  }
  return conversationMemory.get(sessionId)
}

// Helper function to add a message to the conversation history
function addToHistory(sessionId, role, content) {
  const history = getConversationHistory(sessionId)
  history.push([role, content])
  
  // Keep only the last maxTurns conversations
  const maxTurns = parseInt(process.env.HISTORY_MESSAGES) // Adjust this value based on your needs
  if (history.length > maxTurns * 2) { // *2 because each turn has user & assistant message
    history.splice(0, 2) // Remove oldest turn (user + assistant messages)
  }
}

// CORS activation
fastify.register(fastifyCors, {
  origin: true,
  methods: ['POST']
})

// Chat endpoint
fastify.post('/chat', async (request, reply) => {

  const { message: userMessage, sessionId = 'default' } = request.body
  fastify.log.info(`Message received for session ${sessionId}: ${userMessage}`)
  // Get conversation history for this session
  const history = getConversationHistory(sessionId)

  // Similarity search
  // Create embeddings from the question  
  let vector = await llmEmbeddings.embedQuery(userMessage)
  //request.log.info(vector)

  const esSearchResponse = await client.search({
    index: process.env.ELASTIC_INDEX,
    body: {
      size: parseInt(process.env.MAX_SIMILARITIES),
      query: {
        script_score: {
          query: { match_all: {} },
          script: {
            source: "cosineSimilarity(params.query_vector, 'embedding') + 1.0",
            params: { query_vector: vector } // Your query vector
          }
        }
      }
    }
  })

  //request.log.info(esSearchResponse.hits.hits)
  // Process the search results
  const similarities = [];
  const hits = esSearchResponse.hits.hits;

  try {
    for (const hit of hits) {
      const docId = hit._id;
      const score = hit._score;
      // console.log(`Document ID: ${docId}, Score: ${score}`);

      const source = hit._source;
      const prompt = source.prompt;
      // console.log(`Prompt: ${prompt}`);

      similarities.push({
        prompt: prompt,
        id: docId,
        score: score
      });
    }
  } catch (error) {
    throw new Error(`😡 Failed to process search response: ${err.message}`);
  }

  request.log.info("📝 Similarities:")
  request.log.info("=====================================")
  request.log.info(similarities)
  request.log.info("=====================================")


  // Join similarities
  let repositoryContent = similarities
  .map(similarity => similarity.prompt)
  .join('\n');

  request.log.info("📕 Joined Similarities:")
  request.log.info("=====================================")
  request.log.info(repositoryContent)
  request.log.info("=====================================")

  // Construct messages array with system instructions, context, history, and new message
  let messages = [
    ...history,
    ["system", systemInstructions],
    //["system", "REPOSITORY TREE:\n"+directoryTree],
    ["system", "CONTENT:\n"+repositoryContent],
    ["user", userMessage]
  ]

  // Set appropriate headers for streaming
  reply.raw.writeHead(200, {
    'Content-Type': 'text/plain',
    'Transfer-Encoding': 'chunked',
    'Cache-Control': 'no-cache',
    'Connection': 'keep-alive',
  })

  try {
    let assistantResponse = ''
    const stream = await llm.stream(messages, {
      numCtx: parseInt(process.env.OPTION_NUM_CTX) || 2048
    })
    
    for await (const chunk of stream) {
      // Accumulate the response
      assistantResponse += chunk.content
      // Send each chunk to the client
      reply.raw.write(chunk.content)
    }

    // Add both user message and assistant response to history
    addToHistory(sessionId, "user", userMessage)
    addToHistory(sessionId, "assistant", assistantResponse)

    // End the response
    reply.raw.end()
  } catch (error) {
    fastify.log.error(error)
    reply.raw.write('Error: ' + error.message)
    reply.raw.end()
  }

})

// Optional: Endpoint to clear conversation history
fastify.post('/clear-history', async (request, reply) => {
  const { sessionId = 'default' } = request.body
  conversationMemory.delete(sessionId)
  return { success: true, message: 'Conversation history cleared' }
})


// Start the HTTP server
let httpPort = process.env.PORT || 5050

const start = async () => {
  try {
    await fastify.listen({ port: httpPort, host: '0.0.0.0' })
    fastify.log.info(`Server started on ${fastify.server.address().port}`)
  } catch (err) {
    fastify.log.error(err)
    process.exit(1)
  }
}

start()
