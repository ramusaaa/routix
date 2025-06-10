package routix

import (
	"testing"

	"github.com/ramusaaa/routix"
)

func BenchmarkStringValidator(b *testing.B) {
	schema := routix.NewStringSchema().Min(3).Max(10).Required()

	for i := 0; i < b.N; i++ {
		_ = schema.Validate("Musa")
	}
}
