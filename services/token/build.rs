fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Compile token service proto
    tonic_build::configure()
        .build_server(true)
        .build_client(true)
        .compile_protos(
            &["../../api/proto/auth/token_service.proto"],
            &["../../api/proto/auth"],
        )?;

    // Compile crypto service proto (client only)
    tonic_build::configure()
        .build_server(false)
        .build_client(true)
        .compile_protos(
            &["proto/crypto_service.proto"],
            &["proto"],
        )?;

    Ok(())
}
