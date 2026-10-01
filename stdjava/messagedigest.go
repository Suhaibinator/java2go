package stdjava

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha3"
	"crypto/sha512"
	"fmt"
	"hash"
	"strings"
)

// MessageDigest implements the stateful digest operations using Go's standard
// cryptographic hashes. Provider selection, cloning and MD2 are not supported.
type MessageDigest struct {
	algorithm          string
	algorithmReference *JavaString
	hash               hash.Hash
}

type NoSuchAlgorithmException struct{ ThrowableBase }

func NewNoSuchAlgorithmException(message string) NoSuchAlgorithmException {
	return NoSuchAlgorithmException{newThrowableBase("NoSuchAlgorithmException", message)}
}
func init() {
	RegisterException("NoSuchAlgorithmException", "Exception")
	RegisterJavaType("MessageDigest", ObjectTypeID)
}
func (*MessageDigest) JavaDynamicTypeID() TypeID { return "MessageDigest" }

func MessageDigestGetInstance(algorithm string) *MessageDigest {
	if StringIsNull(algorithm) {
		panic(NewNullPointerException("null algorithm name"))
	}
	var h hash.Hash
	switch strings.ToUpper(algorithm) {
	case "MD5":
		h = md5.New()
	case "SHA", "SHA1", "SHA-1":
		h = sha1.New()
	case "SHA224", "SHA-224":
		h = sha256.New224()
	case "SHA256", "SHA-256":
		h = sha256.New()
	case "SHA384", "SHA-384":
		h = sha512.New384()
	case "SHA512", "SHA-512":
		h = sha512.New()
	case "SHA512/224", "SHA-512/224":
		h = sha512.New512_224()
	case "SHA512/256", "SHA-512/256":
		h = sha512.New512_256()
	case "SHA3-224":
		h = sha3.New224()
	case "SHA3-256":
		h = sha3.New256()
	case "SHA3-384":
		h = sha3.New384()
	case "SHA3-512":
		h = sha3.New512()
	default:
		panic(NewNoSuchAlgorithmException(algorithm + " MessageDigest not available"))
	}
	return &MessageDigest{algorithm: algorithm, hash: h}
}
func (d *MessageDigest) GetAlgorithm() string   { return d.algorithm }
func (d *MessageDigest) GetDigestLength() int32 { return int32(d.hash.Size()) }
func (d *MessageDigest) Reset()                 { d.hash.Reset() }

// Update accepts the Java byte, byte[], byte[] range, and ByteBuffer overloads.
// ByteBuffer consumption changes position only after all remaining bytes enter
// the hash; the backing array and limit retain their identity and value.
func (d *MessageDigest) Update(input any, bounds ...int32) {
	var elements []int8
	switch input := input.(type) {
	case int8:
		_, _ = d.hash.Write([]byte{byte(input)})
		return
	case *PrimitiveArray[int8]:
		if input == nil {
			if len(bounds) == 2 {
				panic(NewIllegalArgumentException("No input buffer given"))
			}
			panic(NewNullPointerException("Cannot read the array length because \"input\" is null"))
		}
		elements = input.Elements
		if len(bounds) == 2 {
			offset, length := bounds[0], bounds[1]
			// Match the JDK public range guard before the provider's check.
			// In particular, zero length accepts a negative offset after
			// that guard, and negative lengths report a provider bounds error.
			if int32(len(elements))-offset < length {
				panic(NewIllegalArgumentException("Input buffer too short"))
			}
			if length == 0 {
				return
			}
			if offset < 0 || length < 0 || int64(offset)+int64(length) > int64(len(elements)) {
				panic(NewArrayIndexOutOfBoundsException(fmt.Sprintf("Range [%d, %d + %d) out of bounds for length %d", offset, offset, length, len(elements))))
			}
			elements = elements[offset : offset+length]
		}
	case *ByteBuffer:
		ReferenceRequireNonNull(input)
		d.Update(input.array, input.offset+input.position, input.Remaining())
		input.position = input.limit
		return
	default:
		if javaReferenceIsNull(input) {
			panic(NewNullPointerException("input"))
		}
		panic(NewIllegalArgumentException("unsupported MessageDigest input"))
	}
	bytes := make([]byte, len(elements))
	for i, v := range elements {
		bytes[i] = byte(v)
	}
	_, _ = d.hash.Write(bytes)
}
func (d *MessageDigest) Digest(input ...*PrimitiveArray[int8]) *PrimitiveArray[int8] {
	if len(input) > 0 {
		d.Update(input[0])
	}
	sum := d.hash.Sum(nil)
	d.hash.Reset()
	result := NewPrimitiveArray[int8](int32(len(sum)), PrimitiveByteTypeID)
	for i, v := range sum {
		result.Elements[i] = int8(v)
	}
	return result
}
