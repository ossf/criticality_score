package signalio

import (
	"encoding/csv"
	"sync"
	"testing"
	"time"

	"github.com/ossf/criticality_score/internal/collector/signal"
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
			if err := w.maybeWriteHeader(); err != nil { // never want an error with these test cases
				t.Errorf("maybeWriteHeader() error = %v", err)
			}
		})
	}
}

func Test_csvWriter_writeRecord(t *testing.T) {
	type fields struct {
		w             *csv.Writer
		header        []string
		headerWritten bool
	}
	tests := []struct { //nolint:govet
		name    string
		fields  fields
		values  map[string]string
		wantErr bool
	}{
		{
			name: "write record with regular error",
			fields: fields{
				w: csv.NewWriter(&mockWriter{
					written: []byte{'a', 'b', 'c'},
					error:   nil,
				}),
				header:        []string{"a", "b", "c"},
				headerWritten: true,
			},
			wantErr: true,
		},
		{
			name: "write record with write error",
			fields: fields{
				w:             &csv.Writer{},
				header:        []string{"a", "b", "c"},
				headerWritten: true,
			},
			wantErr: true,
		},
		{
			name: "write record with maybeWriteHeader error",
			fields: fields{
				w:             &csv.Writer{},
				header:        []string{"a", "b", "c"},
				headerWritten: false,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &csvWriter{
				w:             tt.fields.w,
				header:        tt.fields.header,
				headerWritten: tt.fields.headerWritten,
				mu:            sync.Mutex{},
			}
			if err := w.writeRecord(tt.values); (err != nil) != tt.wantErr {
				t.Errorf("writeRecord() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type mockWriter struct { //nolint:govet
	written []byte
	error   error
}

func (m *mockWriter) Write(p []byte) (n int, err error) {
	return 0, m.error
}

func Test_csvWriter_WriteSignals(t *testing.T) {
	type args struct {
		signals []signal.Set
		extra   []Field
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "write signals with marshal error",
			args: args{
				signals: []signal.Set{
					&testSet{
						UpdatedCount: signal.Val(1),
					},
				},
				extra: []Field{
					{
						Key:   "a",
						Value: []int{1, 2, 3},
					},
					{
						Key:   "b",
						Value: map[string]string{"key": "value"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "write signals with write error",
			args: args{
				extra: []Field{
					{
						Key:   "a",
						Value: "1",
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := CSVWriter(&mockWriter{}, []signal.Set{}, "a", "b")

			if err := writer.WriteSignals(tt.args.signals, tt.args.extra...); (err != nil) != tt.wantErr {
				t.Errorf("WriteSignals() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type testSet struct { //nolint:govet
	UpdatedCount signal.Field[int]
	Field        string
}

func (t testSet) Namespace() signal.Namespace {
	return "test"
}
