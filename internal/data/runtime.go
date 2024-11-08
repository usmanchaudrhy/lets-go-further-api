package data

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Runtime int32

var ErrInvalidRuntimeFormat = errors.New("invalid runtime format")

// Implement the MarshalJSON method on the runtime type
// so that it satisfies the json.Marshal interface
func (r Runtime) MarshalJSON() ([]byte, error) {
	jsonValue := fmt.Sprintf("%d mins", r)

	// we can use the strconv.Quote function to string
	// to wrap it in double quotes
	quotedJSONValue := strconv.Quote(jsonValue)

	return []byte(quotedJSONValue), nil
}

// Go has the JSON.unmarshaler interface. When decoding JSON, Go checks if the value satisfie the
// interface, then Go will call its unmarshalJSON method to determine how to decode the provided
// information. This is basically the reverse of the MarshalJSON mmethod
// Because JSON.UnmarshalJSON() needs to modify the receiver, we must use a pointer for this to work properly.
// Otherwise we will only be modifying the copy
func (r *Runtime) UnmarshalJSON(jsonValue []byte) error {
	// we know that the incoming value will be of the type "<runtime> mins"
	// first we need to remove the quotes. If we can't unquote it we return ErrInvalidRuntimeFormat
	unqotedJSONValue, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		return ErrInvalidRuntimeFormat
	}

	// split the string to isolate the part containing the number
	parts := strings.Split(unqotedJSONValue, " ")
	if len(parts) != 2 || parts[1] != "mins" {
		return ErrInvalidRuntimeFormat
	}

	i, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return ErrInvalidRuntimeFormat
	}

	*r = Runtime(i)
	return nil
}
