package api

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type adfNode = map[string]interface{}

func markdownToADF(src string) adfNode {
	md := goldmark.New()
	reader := text.NewReader([]byte(src))
	doc := md.Parser().Parse(reader)

	content := walkBlock(doc, []byte(src))
	return adfNode{
		"type":    "doc",
		"version": 1,
		"content": content,
	}
}

func walkBlock(node ast.Node, src []byte) []interface{} {
	var nodes []interface{}
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		n := convertBlock(child, src)
		if n != nil {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

func convertBlock(node ast.Node, src []byte) interface{} {
	switch node.Kind() {
	case ast.KindHeading:
		h := node.(*ast.Heading)
		return adfNode{
			"type":    "heading",
			"attrs":   adfNode{"level": h.Level},
			"content": walkInline(node, src),
		}

	case ast.KindParagraph, ast.KindTextBlock:
		return adfNode{
			"type":    "paragraph",
			"content": walkInline(node, src),
		}

	case ast.KindFencedCodeBlock, ast.KindCodeBlock:
		var sb strings.Builder
		for i := 0; i < node.Lines().Len(); i++ {
			line := node.Lines().At(i)
			sb.Write(line.Value(src))
		}
		code := strings.TrimRight(sb.String(), "\n")
		var langAttr adfNode
		if fc, ok := node.(*ast.FencedCodeBlock); ok && fc.Info != nil {
			lang := string(fc.Info.Segment.Value(src))
			langAttr = adfNode{"language": lang}
		} else {
			langAttr = adfNode{}
		}
		return adfNode{
			"type":  "codeBlock",
			"attrs": langAttr,
			"content": []interface{}{
				adfNode{"type": "text", "text": code},
			},
		}

	case ast.KindList:
		l := node.(*ast.List)
		if l.IsOrdered() {
			return adfNode{
				"type":    "orderedList",
				"attrs":   adfNode{"order": l.Start},
				"content": walkBlock(node, src),
			}
		}
		return adfNode{
			"type":    "bulletList",
			"content": walkBlock(node, src),
		}

	case ast.KindListItem:
		return adfNode{
			"type":    "listItem",
			"content": walkBlock(node, src),
		}

	case ast.KindBlockquote:
		return adfNode{
			"type":    "blockquote",
			"content": walkBlock(node, src),
		}

	case ast.KindThematicBreak:
		return adfNode{"type": "rule"}
	}

	return nil
}

func walkInline(node ast.Node, src []byte) []interface{} {
	var nodes []interface{}
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		collected := convertInline(child, src, nil)
		nodes = append(nodes, collected...)
	}
	return nodes
}

func convertInline(node ast.Node, src []byte, marks []interface{}) []interface{} {
	switch node.Kind() {
	case ast.KindText:
		t := node.(*ast.Text)
		txt := string(t.Segment.Value(src))
		n := adfNode{"type": "text", "text": txt}
		if len(marks) > 0 {
			n["marks"] = marks
		}
		result := []interface{}{n}
		if t.SoftLineBreak() || t.HardLineBreak() {
			result = append(result, adfNode{"type": "hardBreak"})
		}
		return result

	case ast.KindString:
		txt := string(node.(*ast.String).Value)
		n := adfNode{"type": "text", "text": txt}
		if len(marks) > 0 {
			n["marks"] = marks
		}
		return []interface{}{n}

	case ast.KindEmphasis:
		em := node.(*ast.Emphasis)
		var markType string
		if em.Level == 2 {
			markType = "strong"
		} else {
			markType = "em"
		}
		newMarks := append(marks, adfNode{"type": markType})
		var result []interface{}
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			result = append(result, convertInline(child, src, newMarks)...)
		}
		return result

	case ast.KindCodeSpan:
		var sb strings.Builder
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if s, ok := child.(*ast.Text); ok {
				sb.Write(s.Segment.Value(src))
			}
		}
		newMarks := append(marks, adfNode{"type": "code"})
		return []interface{}{adfNode{"type": "text", "text": sb.String(), "marks": newMarks}}

	case ast.KindLink:
		lnk := node.(*ast.Link)
		href := string(lnk.Destination)
		newMarks := append(marks, adfNode{"type": "link", "attrs": adfNode{"href": href}})
		var result []interface{}
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			result = append(result, convertInline(child, src, newMarks)...)
		}
		return result

	case ast.KindAutoLink:
		al := node.(*ast.AutoLink)
		url := string(al.URL(src))
		mark := adfNode{"type": "link", "attrs": adfNode{"href": url}}
		allMarks := append(marks, mark)
		return []interface{}{adfNode{"type": "text", "text": url, "marks": allMarks}}

	case ast.KindRawHTML:
		return nil
	}

	// fallback: recurse
	var result []interface{}
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		result = append(result, convertInline(child, src, marks)...)
	}
	return result
}
