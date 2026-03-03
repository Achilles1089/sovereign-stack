// swift-tools-version: 5.9
import PackageDescription

let llamaVersion = "b6871"

let package = Package(
    name: "SovereignAI",
    platforms: [
        .iOS(.v17),
        .macOS(.v14)
    ],
    products: [
        .library(
            name: "SovereignAI",
            targets: ["SovereignAI"]
        ),
    ],
    dependencies: [
        // Vapor HTTP server framework
        .package(url: "https://github.com/vapor/vapor.git", from: "4.89.0"),
    ],
    targets: [
        // Prebuilt llama.cpp XCFramework from official releases
        .binaryTarget(
            name: "llama-xcframework",
            url: "https://github.com/ggml-org/llama.cpp/releases/download/\(llamaVersion)/llama-\(llamaVersion)-xcframework.zip",
            checksum: "ac657d70112efadbf5cd1db5c4f67eea94ca38556ada9e7442d5a5a461010d6f"
        ),
        .target(
            name: "SovereignAI",
            dependencies: [
                .product(name: "Vapor", package: "vapor"),
                "llama-xcframework",
            ],
            path: "SovereignAI",
            linkerSettings: [
                .linkedFramework("Metal"),
                .linkedFramework("MetalKit"),
                .linkedFramework("Accelerate"),
            ]
        ),
    ]
)
