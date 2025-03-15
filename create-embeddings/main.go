package main

import (
	"context"
	"embeddings-gitingest/content"
	"embeddings-gitingest/embeddings"
	"embeddings-gitingest/helpers"
	"embeddings-gitingest/source"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/ollama/ollama/api"
)

func main() {
	ctx := context.Background()

	contentPath := helpers.GetEnvString("CONTENT_PATH", "../data/content.txt")

	contentPrefixForId := helpers.GetEnvString("CONTENT_PREFIX_FOR_ID", "content")

	ollamaUrl := helpers.GetEnvString("OLLAMA_BASE_URL", "http://localhost:11434")

	url, _ := url.Parse(ollamaUrl)

	ollamaClient := api.NewClient(url, http.DefaultClient)

	embeddingsModel := helpers.GetEnvString("LLM_EMBEDDINGS", "mxbai-embed-large")

	elasticStore := embeddings.ElasticsearchStore{}
	err := elasticStore.Initialize(
		[]string{
			os.Getenv("ELASTICSEARCH_HOSTS"),
		},
		os.Getenv("ELASTICSEARCH_USERNAME"),
		os.Getenv("ELASTICSEARCH_PASSWORD"),
		nil,
		os.Getenv("ELASTICSEARCH_INDEX"),
	)
	if err != nil {
		log.Fatalln("😡:", err)
	}

	// open ../data/content.txt
	// read the content
	allSourceCodes, err := os.ReadFile(contentPath)
	if err != nil {
		log.Fatal(err)
	}

	// First pass: Split the Gitingest content into chunks
	// Ok, it's not my best idea to use this delimiter
	chunksFromAllSourceCodes := content.SplitTextWithDelimiter(
		string(allSourceCodes),
		`================================================`,
	)

	// Second pass: Extract code elements from the chunk and create embeddings
	for idx, chunk := range chunksFromAllSourceCodes {

		fmt.Println("🔍 Extracting code elements...")
		// For example, extract the function signatures
		elements, err := source.ExtractCodeElements(chunk, "go")
		if err != nil {
			fmt.Println("😡 when extracting element:", err)
			continue
		}
		header := "METADATA:\n"
		// use the function signatures as metadata
		for _, element := range elements {
			header += element.Signature + "\n"
			fmt.Println("📝", element.Signature)
		}
		fmt.Println("================================================")
		// Create the embeddings

		req := &api.EmbeddingRequest{
			Model:  embeddingsModel,
			Prompt: header + chunk,
		}
		// get embeddings
		resp, err := ollamaClient.Embeddings(ctx, req)
		if err != nil {
			log.Println("😡:", err)

		}
		embedding := embeddings.VectorRecord{
			Id:        contentPrefixForId + "-" + strconv.Itoa(idx),
			Prompt:    header + chunk,
			Embedding: resp.Embedding,
		}

		if err != nil {
			fmt.Println("😡 when generating embedding:", err)
		} else {
			if _, err = elasticStore.Save(embedding); err != nil {
				log.Fatalln("😡 we have a problem with ES when saving embedding:", err)
			}
			fmt.Println("🎉 Document", embedding.Id, "indexed successfully")
		}
	}

}
