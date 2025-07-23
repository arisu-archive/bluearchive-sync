package javax

import (
	"fmt"
	"math/bits"
	"reflect"
)

// HashMap represents java.util.HashMap - generic key-value map
type HashMap struct {
	LoadFactor float32     `javaio:"loadFactor"`
	Threshold  int32       `javaio:"threshold"`
	Data       map[any]any `javaio:"-"` // The actual map data with type safety
}

func NewHashMap(data map[any]any) *HashMap {
	return &HashMap{
		LoadFactor: 0.75,
		Threshold:  1,
		Data:       data,
	}
}

func (HashMap) ClassName() string {
	return "java.util.HashMap"
}

func (HashMap) SerialVersionUID() int64 {
	return 362498820763181265 // From the parsed data
}

// Custom read method to handle the map data
func (h *HashMap) ReadObject(dec *Decoder) error {
	// First read the default fields
	if err := dec.DefaultReadFields(); err != nil {
		return err
	}

	// Initialize the maps
	h.Data = make(map[any]any)

	// Read the block data - this contains capacity and size info
	var capacity, size int32
	if err := dec.ReadBinary(&capacity, &size); err != nil {
		return err
	}

	// Read the key-value pairs
	for range size {
		// Read key (can be any object)
		keyObj, err := dec.ReadObject()
		if err != nil {
			return fmt.Errorf("failed to read key: %w", err)
		}

		// Read value (can be any object)
		valueObj, err := dec.ReadObject()
		if err != nil {
			return fmt.Errorf("failed to read value: %w", err)
		}

		// Handle potential pointer dereferencing for key
		key, err := toGoType(keyObj)
		if err != nil {
			return fmt.Errorf("failed to assert key type: %w", err)
		}

		// Handle potential pointer dereferencing for value
		value, err := toGoType(valueObj)
		if err != nil {
			return fmt.Errorf("failed to assert value type: %w", err)
		}
		h.Data[key] = value
	}

	return nil
}

func (h *HashMap) SerializeObject(enc *Encoder) error {
	// Write TcObject
	if err := enc.writeBinary(TcObject); err != nil {
		return err
	}

	// Write TcClassdesc
	if err := enc.writeBinary(TcClassdesc); err != nil {
		return err
	}

	// Write class descriptor
	className := h.ClassName()
	if err := enc.writeUTF(className); err != nil {
		return err
	}

	// Write serialVersionUID
	if err := enc.writeBinary(h.SerialVersionUID()); err != nil {
		return err
	}

	// Write handle for the class descriptor holder
	enc.newHandle(enc.classNameHolder(h.ClassName()))

	// Write classDescInfo - this includes the missing ObjectAnnotation
	if err := enc.classDescInfo(h); err != nil {
		return err
	}

	// Write handle for the object instance
	enc.newHandle(h)

	// Write the actual field values
	if err := enc.writeBinary(h.LoadFactor); err != nil {
		return err
	}
	if err := enc.writeBinary(h.Threshold); err != nil {
		return err
	}

	// Write block data for HashMap-specific serialization
	if err := enc.writeBinary(TcBlockdata); err != nil {
		return err
	}

	if err := h.WriteObject(enc); err != nil {
		return err
	}

	return enc.writeBinary(TcEndblockdata)
}

func (h *HashMap) WriteObject(enc *Encoder) error {
	// Calculate size
	size := int32(len(h.Data))

	// Write block data size (8 bytes: capacity + size)
	if err := enc.writeBinary(byte(8)); err != nil {
		return err
	}

	// Write capacity and size (matching what ReadObject expects)
	capacity := calculateCapacity(size, h.LoadFactor)
	if err := enc.writeBinary(int32(capacity), int32(size)); err != nil {
		return err
	}

	// Write the key-value pairs
	for key, value := range h.Data {
		// Write key
		if err := enc.WriteObject(key); err != nil {
			return fmt.Errorf("failed to write key: %w", err)
		}

		// Write value
		if err := enc.WriteObject(value); err != nil {
			return fmt.Errorf("failed to write value: %w", err)
		}
	}

	return nil
}

func calculateCapacity(size int32, loadFactor float32) int32 {
	capacity := int32(16)
	if size > 0 {
		minCapacity := int32(float32(size) / loadFactor)
		capacity = nextPowerOfTwo(minCapacity)
	}
	return capacity
}

func nextPowerOfTwo(value int32) int32 {
	if value <= 0 {
		return 1
	}
	return 1 << (32 - bits.LeadingZeros32(uint32(value)))
}

// GetTypedEntry safely retrieves a typed key-value pair from the HashMap
func (h *HashMap) Get(key any) (any, bool) {
	value, exists := h.Data[key]
	return value, exists
}

func (h *HashMap) Set(key any, value any) {
	h.Data[key] = value
}

// Size returns the number of entries in the HashMap
func (h *HashMap) Size() int {
	return len(h.Data)
}

func toGoType(obj any) (any, error) {
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	// Based on the type of the object, return the corresponding Go type
	// We only need to handle the case where the object is NOT a known Go type.
	// In that case, we check if the type is javaio.String object.
	// Check if the object is a javaio.String / javax.Integer / javax.Number. If matched, return the object.Value.
	switch val := rv.Interface().(type) {
	case String:
		return val.Value, nil
	case Integer:
		return val.Value, nil
	}
	return rv.Interface(), nil
}

// toJavaType converts Go types to Java types for serialization
func toJavaType(obj any) (any, error) {
	switch val := obj.(type) {
	case string:
		return String{Value: val}, nil
	case int:
		return &Integer{Value: int32(val)}, nil
	case int32:
		return &Integer{Value: val}, nil
	case int64:
		return &Integer{Value: int32(val)}, nil
	default:
		// For other types, return as-is (they should already be Java types)
		return obj, nil
	}
}
