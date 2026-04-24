// swift-tools-version:5.9
import PackageDescription

let package = Package(
    name: "Chronoscope",
    platforms: [
        .macOS(.v12)
    ],
    products: [
        .library(
            name: "Chronoscope",
            targets: ["Chronoscope"]
        )
    ],
    dependencies: [
        .package(url: "https://github.com/apple/swift-protobuf.git", from: "1.25.0")
    ],
    targets: [
        .systemLibrary(
            name: "ChronoscopePrivacyC",
            path: "Sources/ChronoscopePrivacyC",
            linkerSettings: [
                .linkedLibrary("chronoscope_privacy"),
                .unsafeFlags(["-L", "../../services/privacy-engine/target/release"])
            ]
        ),
        .target(
            name: "Chronoscope",
            dependencies: [
                .product(name: "SwiftProtobuf", package: "swift-protobuf"),
                "ChronoscopePrivacyC"
            ],
            path: "Sources/Chronoscope"
        ),
        .testTarget(
            name: "ChronoscopeTests",
            dependencies: ["Chronoscope"],
            path: "Tests"
        )
    ]
)
