package character

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	bytes2 "github.com/dcrodman/archon/internal/core/bytes"
	"github.com/dcrodman/archon/internal/core/prs"
	"github.com/go-test/deep"
)

func TestPRS(t *testing.T) {
	decompressedFile := "./testdata/decompressed_stats_file.prs"
	wantDecompressed, err := os.ReadFile(decompressedFile)
	if err != nil {
		t.Fatalf("error opening decompressed_stats_file.prs: %v", err)
	}

	characterStatsFile := "./testdata/wantedCharacterStats.json"
	wantCharacterStats, err := os.ReadFile(characterStatsFile)
	if err != nil {
		t.Fatalf("error opening wantedCharacterStats.json: %v", err)
	}

	testFile := "../../setup/parameters/PlyLevelTbl.prs"
	wantCompressed, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("error opening parameters/PlyLevelTbl.prs: %v", err)
	}

	size, err := prs.DecompressSize(wantCompressed)
	if err != nil {
		t.Fatalf("decompress size err: %v", err)
	}

	gotDecompressed, err := prs.Decompress(wantCompressed, size)
	if err != nil {
		t.Fatalf("decompress err: %v", err)
	}

	if !bytes.Equal(gotDecompressed, wantDecompressed) {
		t.Fatalf("decompressed file does not match expected output")
	}

	stats := [NumCharacterClasses]stats{}
	// Base character class stats are stored sequentially, each 14 bytes long.
	for i := 0; i < 12; i++ {
		bytes2.StructFromBytes(gotDecompressed[i*14:], &stats[i])
	}

	gotCharacterStats, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if s := deep.Equal(gotCharacterStats, wantCharacterStats); len(s) > 0 {
		t.Fatal(s)
	}
}
