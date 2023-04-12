use serde::{Deserialize, Serialize};

#[tokio::main]
async fn main() {
    let key = if get_apikey().is_none() {
        panic!("the OPENAI_API_KEY environment variable is not set")
    };
    let key = key.unwrap();

    loop {
        println!("enter your question, and type ENTER");

        let mut question = String::new();
        std::io::stdin()
            .read_line(&mut question)
            .expect("failed to read line");

        match request(question.trim_end(), key.as_str()).await {
            Ok(v) => println!("Got a reply: {:?}", v),
            Err(e) => println!("Got an error: {:?}", e),
        }
    }
}

fn get_apikey() -> Option<String> {
    match std::env::var("OPENAI_API_KEY") {
        Ok(s) => Some(s),
        Err(_) => None,
    }
}

async fn request(_question: &str, _key: &str) -> Result<(), reqwest::Error> {
    /* something like ...
    let header = String::from("Bearer ");
    let body = Body {
        ...
    };

    let response = reqwest::Client::new()
        .post("https://api.openai.com/v1/chat/completions")
        .header("Authorization", header + key)
        .json(&body)
        .send()
        .await?
        .json(&...)
        .await?;
    */
    Ok(())
}

#[derive(Debug, Serialize, Deserialize)]
struct Body {
    model: String,
    messages: Vec<Message>,
}

#[derive(Debug, Serialize, Deserialize)]
struct Message {
    role: String,
    content: String,
}

#[derive(Debug, Serialize, Deserialize)]
struct Choices {
    message: Message,
    finish_reason: String,
    index: u32,
}

#[derive(Debug, Serialize, Deserialize)]
struct Usage {
    prompt_tokens: u32,
    completion_tokens: u32,
    total_tokens: u32,
}

#[derive(Debug, Serialize, Deserialize)]
struct Response {
    id: String,
    object: String,
    model: String,
    usage: Usage,
}
