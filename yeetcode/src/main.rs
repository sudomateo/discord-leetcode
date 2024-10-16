use dropshot::endpoint;
use dropshot::ApiDescription;
use dropshot::Body;
use dropshot::ConfigDropshot;
use dropshot::ConfigLogging;
use dropshot::ConfigLoggingLevel;
use dropshot::HttpError;
use dropshot::HttpServerStarter;
use dropshot::RequestContext;
use dropshot::RequestInfo;
use dropshot::UntypedBody;
use ed25519_dalek::Signature;
use ed25519_dalek::Verifier;
use ed25519_dalek::VerifyingKey;
use ed25519_dalek::PUBLIC_KEY_LENGTH;
use ed25519_dalek::SIGNATURE_LENGTH;
use hyper::Response;
use hyper::StatusCode;
use schemars::JsonSchema;
use serde::Deserialize;
use serde::Serialize;
use std::io::Write;

// TODO: Store HTTP client in here.
struct Context {
    public_key: String,
}

// TODO: Implement more of the interaction data type:
// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object
#[derive(Serialize, Deserialize, JsonSchema, Debug)]
struct InteractionRequest {
    id: String,
    application_id: String,
    r#type: isize,
    token: String,
}

#[tokio::main]
async fn main() -> Result<(), String> {
    let public_key = std::env::var("DISCORD_PUBLIC_KEY").map_err(|error| {
        format!(
            "environment variable DISCORD_PUBLIC_KEY is required: {}",
            error
        )
    })?;

    let config_dropshot: ConfigDropshot = ConfigDropshot {
        bind_address: "0.0.0.0:3000".parse().unwrap(),
        request_body_max_bytes: 8192,
        ..Default::default()
    };

    let config_logging = ConfigLogging::StderrTerminal {
        level: ConfigLoggingLevel::Info,
    };

    let logger = config_logging
        .to_logger("yeetcode")
        .map_err(|error| format!("failed to create logger: {}", error))?;

    let mut api = ApiDescription::new();
    api.register(interaction_handler).unwrap();

    let ctx = Context { public_key };

    let server = HttpServerStarter::new(&config_dropshot, api, ctx, &logger)
        .map_err(|error| format!("failed to create server: {}", error))?
        .start();

    server.await
}

// TODO: Implement more Rust-y error handling.
#[endpoint {
    method = POST,
    path = "/",
}]
async fn interaction_handler(
    ctx: RequestContext<Context>,
    body: UntypedBody,
) -> Result<Response<Body>, HttpError> {
    let req_ctx = ctx.context();

    let public_key: [u8; PUBLIC_KEY_LENGTH] = hex::decode(&req_ctx.public_key)
        .unwrap()
        .try_into()
        .unwrap();

    verify_interaction(&ctx.request, &body, &public_key).map_err(|error| HttpError {
        status_code: StatusCode::UNAUTHORIZED,
        error_code: Some(String::from("verify_interaction")),
        external_message: format!("failed to verify interaction: {}", error),
        internal_message: format!("failed to verify interaction: {}", error),
    })?;

    let interaction_req: InteractionRequest = serde_json::from_slice(&body.as_bytes()).unwrap();

    if interaction_req.r#type == 1 {
        return Ok(Response::builder()
            .status(StatusCode::OK)
            .header("Content-Type", "application/json")
            .body(r#"{"type": 1}"#.into())?);
    }

    dbg!(&interaction_req);

    let client = reqwest::Client::new();

    // TODO: Create a LeetCode client to fetch a random question.
    let _ = client
        .post(format!(
            "https://discord.com/api/v10/interactions/{}/{}/callback",
            &interaction_req.id, &interaction_req.token
        ))
        .header("Content-Type", "application/json")
        .body(r#"{"type": 4, "data": {"content": "https://leetcode.com/problems/two-sum"}}"#)
        .send()
        .await
        .map_err(|error| HttpError {
            status_code: StatusCode::INTERNAL_SERVER_ERROR,
            error_code: Some(String::from("post_callback")),
            external_message: format!("failed to post callback: {}", error),
            internal_message: format!("failed to post callback: {}", error),
        });

    Ok(Response::builder()
        .status(StatusCode::OK)
        .header("Content-Type", "application/json")
        .body("OK".into())?)
}

fn verify_interaction(
    request: &RequestInfo,
    body: &UntypedBody,
    public_key: &[u8; PUBLIC_KEY_LENGTH],
) -> Result<(), Box<dyn std::error::Error>> {
    let verifying_key = VerifyingKey::from_bytes(&public_key)?;

    let headers = request.headers();

    let signature_bytes = headers.get("X-Signature-Ed25519").unwrap().as_bytes();
    let signature_bytes: [u8; SIGNATURE_LENGTH] =
        hex::decode(signature_bytes).unwrap().try_into().unwrap();

    let signature = Signature::from_bytes(&signature_bytes);

    let timestamp_bytes = headers.get("X-Signature-Timestamp").unwrap().as_bytes();

    let mut buf: Vec<u8> = Vec::new();
    buf.write(timestamp_bytes).expect("write_timestamp");
    buf.write(body.as_bytes()).expect("write_message");

    Ok(verifying_key.verify(&buf, &signature)?)
}

// #[cfg(test)]
// mod tests {
//     use super::*;
//
//     #[test]
//     fn it_verifies() {
//         let public_key_bytes: [u8; PUBLIC_KEY_LENGTH] =
//             hex::decode("7a111dd70fe28562fa15c7d0d26b06dd48bdfe58e73397576d2e77d1ec8db01c")
//                 .unwrap()
//                 .try_into()
//                 .unwrap();
//
//         let signature_bytes: [u8; SIGNATURE_LENGTH] =
//             hex::decode("f9c58d011a41280b7ec0d2bf13bef8b3f8de4933d26c304e2e8ef00ca23d6ba3739ba5cf70a5b3fc1cda1c271a16d0162db514000d2fcf1eca155815ebb2c904")
//                 .unwrap()
//                 .try_into()
//                 .unwrap();
//
//         let timestamp_bytes = b"1728695487";
//         let message = br#"{"app_permissions":"562949953601536","application_id":"1293749125078188085","authorizing_integration_owners":{},"entitlements":[],"id":"1294467436514381885","token":"aW50ZXJhY3Rpb246MTI5NDQ2NzQzNjUxNDM4MTg4NTphN2lCWk1xbnNLRkhobXFwNVBFa2ZGcHBnZTFGOEZ5QjJ1UGhEaHJMc2g0d0k0a0dlY0duV1JDZk4wRGhJZHdISWdGVE5Cd0x6NzZ1cXNJUFNjTkNzMDA1NFQ3alo5aExwR2hFM2NYNUJPWDJWT2QwM2JFd3IxYmNmQmdlUVBNTw","type":1,"user":{"avatar":"c6a249645d46209f337279cd2ca998c7","avatar_decoration_data":null,"bot":true,"clan":null,"discriminator":"0000","global_name":"Discord","id":"643945264868098049","public_flags":1,"system":true,"username":"discord"},"version":1}"#;
//
//         assert!(verify(
//             &public_key_bytes,
//             &signature_bytes,
//             timestamp_bytes,
//             message
//         ));
//     }
//
//     #[test]
//     fn it_verifies_again() {
//         let public_key_bytes: [u8; PUBLIC_KEY_LENGTH] =
//             hex::decode("7a111dd70fe28562fa15c7d0d26b06dd48bdfe58e73397576d2e77d1ec8db01c")
//                 .unwrap()
//                 .try_into()
//                 .unwrap();
//
//         let signature_bytes: [u8; SIGNATURE_LENGTH] =
//             hex::decode("b5c9cb40580bd3f35a4d75a37243b9e2cc5c75e83c41b10f3aa08163245e581f854e62b027104cab3bc8d8da5bdc9a484520905beb051c01a88201d31c9d9705")
//                 .unwrap()
//                 .try_into()
//                 .unwrap();
//
//         let timestamp_bytes = b"1728695486";
//         let message = br#"{"app_permissions":"562949953601536","application_id":"1293749125078188085","authorizing_integration_owners":{},"entitlements":[],"id":"1294467436514381886","token":"aW50ZXJhY3Rpb246MTI5NDQ2NzQzNjUxNDM4MTg4NjpQaTYwakVqYkFzSU5JZjVoSVpWVmpUUTVaTHhHQmpXUDFwUDdIcDF6bkx6NmJLN0hRWUlobHVTeE5STmJBUUM2NlAyWWZCYmptSXlGc3dpR3lxVWJXMXNIR0hITVVHRnBlOE5XZTlLeXFLS1JCRW85bGpsS1dyMmExTkV3ekpocg","type":1,"user":{"avatar":"c6a249645d46209f337279cd2ca998c7","avatar_decoration_data":null,"bot":true,"clan":null,"discriminator":"0000","global_name":"Discord","id":"643945264868098049","public_flags":1,"system":true,"username":"discord"},"version":1}"#;
//
//         assert!(!verify(
//             &public_key_bytes,
//             &signature_bytes,
//             timestamp_bytes,
//             message
//         ));
//     }
// }
