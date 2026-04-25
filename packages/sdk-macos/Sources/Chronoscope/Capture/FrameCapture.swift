import Foundation
import ScreenCaptureKit
import CoreImage
import CoreVideo
import CoreMedia
import AppKit
import OSLog

private let logger = Logger(subsystem: "dev.chronoscope.sdk", category: "FrameCapture")

/// Captures screen frames using ScreenCaptureKit.
public actor FrameCapture: NSObject {
    private var stream: SCStream?
    private var frameHandler: ((Data) async -> Void)?
    private let frameRate: Int
    private let privacyEngine: PrivacyEngine?
    private var onError: ((Error) -> Void)?

    /// Creates a new frame capture instance.
    /// - Parameters:
    ///   - frameRate: Target frame rate. Defaults to `10`.
    ///   - privacyEngine: Optional privacy engine for frame redaction.
    public init(frameRate: Int = 10, privacyEngine: PrivacyEngine? = nil) {
        self.frameRate = frameRate
        self.privacyEngine = privacyEngine
        super.init()
    }

    /// Starts capturing frames.
    /// - Parameters:
    ///   - handler: Async closure called for each captured frame.
    ///   - onError: Optional closure called when capture fails.
    public func start(
        handler: @escaping (Data) async -> Void,
        onError: ((Error) -> Void)? = nil
    ) async {
        self.frameHandler = handler
        self.onError = onError

        do {
            let content = try await SCShareableContent.current
            guard let display = content.displays.first else {
                logger.error("No display found for capture")
                onError?(CaptureError.noDisplay)
                return
            }

            let filter = SCContentFilter(display: display, excludingWindows: [])
            let configuration = SCStreamConfiguration()
            configuration.width = Int(display.width)
            configuration.height = Int(display.height)
            configuration.minimumFrameInterval = CMTime(value: 1, timescale: CMTimeScale(frameRate))
            configuration.queueDepth = 3

            let newStream = SCStream(filter: filter, configuration: configuration, delegate: self)
            try newStream.addStreamOutput(self, type: .screen, sampleHandlerQueue: .global(qos: .userInitiated))
            try await newStream.startCapture()
            self.stream = newStream
        } catch {
            logger.error("Failed to start frame capture: \(error.localizedDescription)")
            onError?(error)
        }
    }

    /// Stops capturing frames and tears down the stream.
    public func stop() async {
        if let stream = stream {
            try? await stream.stopCapture()
        }
        stream = nil
        frameHandler = nil
        onError = nil
    }

    deinit {
        if stream != nil {
            logger.warning("FrameCapture deallocated without calling stop()")
        }
    }
}

extension FrameCapture: SCStreamDelegate {
    nonisolated public func stream(_ stream: SCStream, didStopWithError error: Error) {
        logger.error("SCStream stopped with error: \(error.localizedDescription)")
        Task { [self] in
            await self.onError?(error)
        }
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

        CVPixelBufferLockBaseAddress(pixelBuffer, .readOnly)
        defer { CVPixelBufferUnlockBaseAddress(pixelBuffer, .readOnly) }

        guard let baseAddress = CVPixelBufferGetBaseAddress(pixelBuffer) else { return }
        let width = Int(CVPixelBufferGetWidth(pixelBuffer))
        let height = Int(CVPixelBufferGetHeight(pixelBuffer))
        let stride = Int(CVPixelBufferGetBytesPerRow(pixelBuffer))
        let frameSize = height * stride

        var frameData = Data(bytes: baseAddress, count: frameSize)

        Task { [self] in
            await self.privacyEngine?.processFrame(
                &frameData,
                width: UInt32(width),
                height: UInt32(height),
                stride: UInt32(stride)
            )
            guard let jpegData = encodeToJPEG(
                rgbaData: frameData,
                width: width,
                height: height,
                stride: stride
            ) else { return }
            await self.frameHandler?(jpegData)
        }
    }
}

private nonisolated func encodeToJPEG(rgbaData: Data, width: Int, height: Int, stride: Int) -> Data? {
    guard let rep = NSBitmapImageRep(
        bitmapDataPlanes: nil,
        pixelsWide: width,
        pixelsHigh: height,
        bitsPerSample: 8,
        samplesPerPixel: 4,
        hasAlpha: true,
        isPlanar: false,
        colorSpaceName: .deviceRGB,
        bytesPerRow: stride,
        bitsPerPixel: 32
    ) else {
        return nil
    }
    guard let bitmapData = rep.bitmapData else { return nil }
    rgbaData.copyBytes(to: bitmapData, count: rgbaData.count)
    return rep.representation(using: .jpeg, properties: [.compressionFactor: 0.7])
}

/// Local errors produced by FrameCapture.
private enum CaptureError: Error {
    case noDisplay
}
