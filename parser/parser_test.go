package parser

import (
	"testing"
)

func TestParser(t *testing.T) {
	p := Parser{}
	p.ParseSourceFile(`
		<?php echo "test";
`, "")
}
