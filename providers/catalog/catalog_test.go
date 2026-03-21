package catalog

import "testing"

func TestRegisterAndLookup(t *testing.T) {
	Register(LanguageInfo{ID: "php", Extensions: []string{".php", ".PHTML", "php5"}})

	if info, ok := LookupByExtension(".php"); !ok || info.ID != "php" {
		t.Fatalf("expected php for .php, got %v %v", info, ok)
	}

	if info, ok := LookupByExtension(".phtml"); !ok || info.ID != "php" {
		t.Fatalf("expected php for .phtml, got %v %v", info, ok)
	}

	if info, ok := LookupByExtension(".php5"); !ok || info.ID != "php" {
		t.Fatalf("expected php for php5, got %v %v", info, ok)
	}

	langs := Languages()
	if len(langs) == 0 {
		t.Fatal("expected languages slice not empty")
	}
}
