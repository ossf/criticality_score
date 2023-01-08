package signalio

import (
	"encoding/csv"
	"sync"
	"testing"
	"time"
)

func TestMarshalValue(t *testing.T) {
	tests := []struct {
		value    any
		expected string
		wantErr  bool
	}{
		{value: true, expected: "true", wantErr: false},
		{value: 1, expected: "1", wantErr: false},
		{value: int16(2), expected: "2", wantErr: false},
		{value: int32(3), expected: "3", wantErr: false},
		{value: int64(4), expected: "4", wantErr: false},
		{value: uint(5), expected: "5", wantErr: false},
		{value: uint16(6), expected: "6", wantErr: false},
		{value: uint32(7), expected: "7", wantErr: false},
		{value: uint64(8), expected: "8", wantErr: false},
		{value: byte(9), expected: "9", wantErr: false},
		{value: float32(10.1), expected: "10.1", wantErr: false},
		{value: 11.1, expected: "11.1", wantErr: false}, // float64
		{value: "test", expected: "test", wantErr: false},
		{value: time.Now(), expected: time.Now().Format(time.RFC3339), wantErr: false},
		{value: nil, expected: "", wantErr: false},
		{value: []int{1, 2, 3}, expected: "", wantErr: true},
		{value: map[string]string{"key": "value"}, expected: "", wantErr: true},
		{value: struct{}{}, expected: "", wantErr: true},
	}
	for _, test := range tests {
		res, err := marshalValue(test.value)
		if (err != nil) != test.wantErr {
			t.Errorf("Unexpected error for value %v: wantErr %v, got %v", test.value, test.wantErr, err)
		}
		if res != test.expected {
			t.Errorf("Unexpected result for value %v: expected %v, got %v", test.value, test.expected, res)
		}
	}
}

func Test_csvWriter_maybeWriteHeader(t *testing.T) {
	type fields struct {
		w             *csv.Writer
		header        []string
		headerWritten bool
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "write header",
			fields: fields{
				w:             csv.NewWriter(nil),
				header:        []string{},
				headerWritten: false,
			},
		},
		{
			name: "header already written",
			fields: fields{
				w:             csv.NewWriter(nil),
				header:        []string{"a", "b", "c"},
				headerWritten: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := &csvWriter{
				w:             test.fields.w,
				header:        test.fields.header,
				headerWritten: test.fields.headerWritten,
				mu:            sync.Mutex{},
			}
			if err := w.maybeWriteHeader(); err != nil { // never want an error
				t.Errorf("maybeWriteHeader() error = %v", err)
			}
		})
	}
}
