// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "SovereignAI",
    platforms: [
        .iOS(.v16),
        .macOS(.v13)
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
        // llama.cpp for Apple platforms (Stanford BDHG maintained fork)
        .package(url: "https://github.com/StanfordBDHG/llama.cpp-spm.git", from: "0.3.3"),
    ],
    targets: [
        .target(
            name: "SovereignAI",
            dependencies: [
                .product(name: "Vapor", package: "vapor"),
                .product(name: "llama", package: "llama.cpp-spm"),
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
