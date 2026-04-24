import Foundation

public actor CircularBuffer {
    private var storage: Data
    private let capacity: Int
    private var writeOffset: Int = 0
    private var readOffset: Int = 0
    private var totalBytes: Int = 0
    private let lock = NSLock()

    public init(capacity: Int) {
        precondition(capacity > 0, "CircularBuffer capacity must be > 0")
        self.capacity = capacity
        self.storage = Data(count: capacity)
    }

    public func write(_ data: Data) {
        lock.lock()
        defer { lock.unlock() }

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

    public func readChunk() -> Data? {
        lock.lock()
        defer { lock.unlock() }

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
