import Foundation
import ScreenCaptureKit
import CoreImage
import CoreVideo
import CoreMedia
import AppKit

public actor FrameCapture: NSObject {
    private var stream: SCStream?
    private var frameHandler: ((Data) -> Void)?
    private let frameRate: Int

    private let privacyEngine: PrivacyEngine?

    public init(frameRate: Int = 10, privacyEngine: PrivacyEngine? = nil) {
        self.frameRate = frameRate
        self.privacyEngine = privacyEngine
        super.init()
    }

    public func start(handler: @escaping (Data) -> Void) async {
        self.frameHandler = handler

        do {
            let content = try await SCShareableContent.current
            guard let display = content.displays.first else {
                print("No display found for capture")
                return
            }

            let filter = SCContentFilter(display: display, excludingWindows: [])
            let configuration = SCStreamConfiguration()
            configuration.width = Int(display.width)
            configuration.height = Int(display.height)
            configuration.minimumFrameInterval = CMTime(value: 1, timescale: CMTimeScale(frameRate))
            configuration.queueDepth = 3

            let newStream = SCStream(filter: filter, configuration: configuration, delegate: nil)
            try newStream.addStreamOutput(self, type: .screen, sampleHandlerQueue: .global(qos: .userInitiated))
            try await newStream.startCapture()
            self.stream = newStream
        } catch {
            print("Failed to start frame capture: \(error)")
        }
    }

    public func stop() async {
        if let stream = stream {
            try? await stream.stopCapture()
        }
        stream = nil
        frameHandler = nil
    }
}

extension FrameCapture: SCStreamOutput {
    nonisolated public func stream(
        _ stream: SCStream,
        didOutputSampleBuffer sampleBuffer: CMSampleBuffer,
        of outputType: SCStreamOutputType
    ) {
        guard outputType == .screen else { return }
        guard let pixelBuffer = sampleBuffer.imageBuffer else { return }

        let ciImage = CIImage(cvPixelBuffer: pixelBuffer)
        let context = CIContext()
        guard let cgImage = context.createCGImage(ciImage, from: ciImage.extent) else { return }
        let nsImage = NSImage(cgImage: cgImage, size: NSSize(width: cgImage.width, height: cgImage.height))

        guard let tiffData = nsImage.tiffRepresentation,
              let bitmap = NSBitmapImageRep(data: tiffData),
              let jpegData = bitmap.representation(using: .jpeg, properties: [.compressionFactor: 0.7]) else {
            return
        }

        // TODO: Apply privacy filtering to raw pixel buffer before JPEG encoding.
        // PrivacyEngine.processFrame requires raw RGBA data, but this pipeline
        // currently produces JPEG. Modifying the full frame handler pipeline is
        // complex and deferred post-MVP.

        Task { [jpegData] in
            await self.frameHandler?(jpegData)
        }
    }
}
