package product

import "testing"

func TestParsePrice(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{name: "rubles", input: "49 990 ₽", want: 4_999_000},
		{name: "kopecks", input: "1 234,50 ₽", want: 123_450},
		{name: "one kopeck digit", input: "10.5", want: 1_050},
		{name: "empty", input: "", wantErr: true},
		{name: "zero", input: "0 ₽", wantErr: true},
		{name: "text", input: "price unavailable", wantErr: true},
		{name: "text with digits", input: "price 100", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePrice(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParsePrice() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ParsePrice() = %d, want %d", got, tt.want)
			}
		})
	}
}
