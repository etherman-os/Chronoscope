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

            let newStream = SCStream(filter: filter, configuration: configuration, delegate: self)
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

extension FrameCapture: SCStreamDelegate {
    nonisolated public func stream(_ stream: SCStream, didStopWithError error: Error) {
        print("SCStream stopped with error: \(error)")
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

        Task {
            await privacyEngine?.processFrame(&frameData, width: UInt32(width), height: UInt32(height), stride: UInt32(stride))
            guard let jpegData = encodeToJPEG(rgbaData: frameData, width: width, height: height, stride: stride) else { return }
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
    rgbaData.copyBytes(to: rep.bitmapData!, count: rgbaData.count)
    return rep.representation(using: .jpeg, properties: [.compressionFactor: 0.7])
}
