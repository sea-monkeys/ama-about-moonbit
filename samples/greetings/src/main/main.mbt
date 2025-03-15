
pub fn greet() -> Int {
  let jsonString = @host.input_string()

  //let jsonString = "{\"name\":\"Bobby\"}"


  let parsedJson = match @json.parse?(jsonString) {
    Ok(jsonValue) => jsonValue
    Err(e) => "unable to parse JSON: \{e}"
  }

  // Accessing fields in the parsed JSON
  let name = match parsedJson {
      { "name": String(n), .. } => n 
      _ => jsonString
  }

  let greeting = "🎉 Greetings, \{name}!"
  
  @host.output_string(greeting)
  0 // success
}

fn main {
  //println(@lib.hello())
}

/// The .. at the end of the pattern tells the compiler that 
/// you're intentionally ignoring any additional fields that might be present in the JSON object. 
/// This approach allows your code to match the "name" field while being flexible about other potential fields in the JSON structure.
