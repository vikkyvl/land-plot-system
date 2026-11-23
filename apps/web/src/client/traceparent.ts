const version = '00'
const sampledFlags = '01'
const traceIdBytes = 16
const spanIdBytes = 8
const hexRadix = 16
const hexDigits = 2

export const traceparentHeader = 'traceparent'

export function newTraceparent(): string {
  return [version, randomHex(traceIdBytes), randomHex(spanIdBytes), sampledFlags].join('-')
}

function randomHex(byteCount: number): string {
  const bytes = new Uint8Array(byteCount)

  do {
    globalThis.crypto.getRandomValues(bytes)
  } while (bytes.every((byte) => byte === 0))

  return Array.from(bytes, (byte) => byte.toString(hexRadix).padStart(hexDigits, '0')).join('')
}
