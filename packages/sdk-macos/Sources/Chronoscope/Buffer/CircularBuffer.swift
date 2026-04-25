import Foundation

/// Thread-safe circular buffer backed by Swift actor isolation.
public actor CircularBuffer {
    private var storage: Data
    private let capacity: Int
    private var writeOffset: Int = 0
    private var readOffset: Int = 0
    private var totalBytes: Int = 0

    /// Creates a new circular buffer with the given capacity.
    /// - Parameter capacity: Byte capacity. Must be greater than 0.
    public init(capacity: Int) {
        precondition(capacity > 0, "CircularBuffer capacity must be > 0")
        self.capacity = capacity
        self.storage = Data(count: capacity)
    }

    /// Writes data into the buffer, overwriting old data when full.
    /// - Parameter data: The data to write.
    public func write(_ data: Data) {
        let bytesToWrite = data.count
        guard bytesToWrite > 0 else { return }

        for i in 0..<bytesToWrite {
            let byte = data[i]
            storage[writeOffset] = byte
            writeOffset = (writeOffset + 1) % capacity
            if totalBytes < capacity {
                totalBytes += 1
            } else {
                readOffset = writeOffset
            }
        }
    }

    /// Reads all available contiguous data from the buffer.
    /// - Returns: The accumulated data, or `nil` if the buffer is empty.
    public func readChunk() -> Data? {
        guard totalBytes > 0 else { return nil }

        var result = Data()
        if readOffset < writeOffset {
            let range = readOffset..<writeOffset
            result.append(storage.subdata(in: range))
            readOffset = writeOffset
        } else {
            let firstPart = storage.subdata(in: readOffset..<capacity)
            let secondPart = storage.subdata(in: 0..<writeOffset)
            result.append(firstPart)
            result.append(secondPart)
            readOffset = writeOffset
        }

        totalBytes = 0
        return result
    }
}
