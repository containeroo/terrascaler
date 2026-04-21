package terraform

import (
	"fmt"
	"math/big"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

type Target struct {
	BlockType string
	Labels    []string
	Attribute string
}

func ReadInt(filename string, src []byte, target Target) (int, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(src, filename)
	if diags.HasErrors() {
		return 0, fmt.Errorf("parse %s: %s", filename, diags.Error())
	}

	block, err := findSyntaxBlock(file.Body, target)
	if err != nil {
		return 0, err
	}

	attr, ok := block.Body.Attributes[target.Attribute]
	if !ok {
		return 0, fmt.Errorf("attribute %q not found in %s", target.Attribute, describeTarget(target))
	}

	value, diags := attr.Expr.Value(nil)
	if diags.HasErrors() {
		return 0, fmt.Errorf("read attribute %q: only literal numeric values are supported: %s", target.Attribute, diags.Error())
	}
	if !value.Type().Equals(cty.Number) {
		return 0, fmt.Errorf("attribute %q is %s, expected number", target.Attribute, value.Type().FriendlyName())
	}

	floatValue := value.AsBigFloat()
	intValue, accuracy := floatValue.Int64()
	if accuracy != big.Exact {
		return 0, fmt.Errorf("attribute %q must be an integer", target.Attribute)
	}
	return int(intValue), nil
}

func SetInt(filename string, src []byte, target Target, value int) ([]byte, error) {
	file, diags := hclwrite.ParseConfig(src, filename, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("parse %s: %s", filename, diags.Error())
	}

	block := findWriteBlock(file.Body(), target)
	if block == nil {
		return nil, fmt.Errorf("%s not found", describeTarget(target))
	}

	block.Body().SetAttributeValue(target.Attribute, cty.NumberIntVal(int64(value)))
	return file.Bytes(), nil
}

func findSyntaxBlock(body hcl.Body, target Target) (*hclsyntax.Block, error) {
	syntaxBody, ok := body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("unsupported HCL body type %T", body)
	}
	for _, block := range syntaxBody.Blocks {
		if block.Type == target.BlockType && labelsEqual(block.Labels, target.Labels) {
			return block, nil
		}
	}
	return nil, fmt.Errorf("%s not found", describeTarget(target))
}

func findWriteBlock(body *hclwrite.Body, target Target) *hclwrite.Block {
	for _, block := range body.Blocks() {
		if block.Type() == target.BlockType && labelsEqual(block.Labels(), target.Labels) {
			return block
		}
	}
	return nil
}

func labelsEqual(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func describeTarget(target Target) string {
	if len(target.Labels) == 0 {
		return fmt.Sprintf("block %q", target.BlockType)
	}
	return fmt.Sprintf("block %q with labels %q", target.BlockType, target.Labels)
}
