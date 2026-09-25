package chapter5

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/net/html"
)

type CountResult struct {
	Words  int
	Images int
}

func CountWordsAndImages(n *html.Node, result *CountResult, skip bool) {
	if n == nil {
		return
	}
	// Count Images
	if n.Type == html.ElementNode && n.Data == "img" {
		result.Images++
	}
	// skip if script or style
	if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
		skip = true
	}
	// Count words
	if n.Type == html.TextNode && !skip {
		text := strings.TrimSpace(n.Data)
		if text != "" {
			// Split by whitespace
			words := strings.Fields(text)
			result.Words += len(words)
		}
	}
	// Recursively process first child
	CountWordsAndImages(n.FirstChild, result, skip)
	// Recursively process next sibling
	CountWordsAndImages(n.NextSibling, result, false)
}

func HtmlParser(url string) {
	body := Fetch(url)

	fmt.Println("HtmlParser")

	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "HtmlParser: %v\n", err)
		return
	}

	forEachNode(doc, startElement, endElement)

	// visit 支持更多的类型
	// link := make([]string, 0)
	// links := visit(link, doc)
	// fmt.Println(links)

	// 查找所有 visiable text
	// texts := make([]string, 0)
	// texts = CollectVisibleText(texts, doc, false)
	// PrintVisibleText(texts)

	// 统计同类element 的数量
	// counts := make(map[string]int)
	// counts = CountElements(counts, doc)
	// PrintElementCounts(counts)

	// 查找所有 link 节点
	// for _, link := range visit(nil, doc) {
	// 	fmt.Println(link)
	// }
}

// visit recursively visits all nodes and collects links
// Plan: Use pure recursion instead of loop to traverse sibling nodes
//  1. Process current node first
//  2. Recursively process first child
//  3. Recursively process next sibling
//
// Tradeoffs: More function calls, but fully recursive as requested
// Risks: None - same result, just different traversal approach
// Updated by hjz on 2026-05-26: Convert loop-based sibling traversal to pure recursion
func visit(links []string, n *html.Node) []string {
	if n == nil {
		return links
	}
	// Check if current node is an anchor tag
	if n.Type == html.ElementNode {
		switch n.Data {
		case "a", "link":
			for _, a := range n.Attr {
				if a.Key == "href" {
					links = append(links, a.Val)
				}
			}
		case "img", "script", "iframe", "video", "audio", "embed":
			for _, a := range n.Attr {
				if a.Key == "src" {
					links = append(links, a.Val)
				}
			}
		default:
			break
		}

	}
	// Recursively process first child
	links = visit(links, n.FirstChild)
	// Recursively process next sibling
	links = visit(links, n.NextSibling)
	return links
}

var depth int = 0

// hasChildren returns true if node has any non-empty child nodes
// Plan: Check if there's at least one child that is not empty text
// Tradeoffs: Simple check works for most cases, some empty elements might still get full tags
// Risks: None - even if wrong, output is still valid HTML
func hasChildren(n *html.Node) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		// If it's an element node, definitely has children
		if c.Type == html.ElementNode {
			return true
		}
		// If it's a text node with non-whitespace content, count as child
		if c.Type == html.TextNode {
			trimmed := strings.TrimSpace(c.Data)
			if trimmed != "" {
				return true
			}
		}
	}
	return false
}

// startElement handles the start of an element (pre-order traversal)
// Plan: Output different node types:
//  1. ElementNode: output opening tag with all attributes
//     - if no children, output self-closing tag <img/>
//     - if has children, output <a href="..."> and increase depth
//  2. CommentNode: output the comment
//  3. TextNode: output the trimmed text content with proper indentation
//
// Tradeoffs: Self-closing heuristic works well for practical purposes
// Risks: None - output is always valid HTML even if heuristic is wrong
// Updated by hjz on 2026-05-26: Exercise 5.7 - complete generic HTML pretty printer
func startElement(n *html.Node) {
	switch n.Type {
	case html.ElementNode:
		// Build opening tag with attributes
		var attrs string
		if len(n.Attr) > 0 {
			for _, attr := range n.Attr {
				attrs += fmt.Sprintf(" %s='%s'", attr.Key, attr.Val)
			}
		}
		// Check if node has children - if not, use self-closing format
		if !hasChildren(n) {
			fmt.Printf("%*s<%s%s/>\n", depth*2, "", n.Data, attrs)
		} else {
			fmt.Printf("%*s<%s%s>\n", depth*2, "", n.Data, attrs)
			depth++
		}
	case html.CommentNode:
		// Output comment with proper indentation
		trimmed := strings.TrimSpace(n.Data)
		if trimmed != "" {
			fmt.Printf("%*s<!-- %s -->\n", depth*2, "", trimmed)
		}
	case html.TextNode:
		// Output text content, skipping empty/whitespace only text
		trimmed := strings.TrimSpace(n.Data)
		if trimmed != "" {
			fmt.Printf("%*s%s\n", depth*2, "", trimmed)
		}
	}
}

// endElement handles the end of an element (post-order traversal)
func endElement(n *html.Node) {
	if n.Type == html.ElementNode && hasChildren(n) {
		depth--
		fmt.Printf("%*s</%s>\n", depth*2, "", n.Data)
	}
}

// forEachNode 针对每个节点 x ， 可选调用 pre(x) , post(x)
func forEachNode(n *html.Node, pre, post func(*html.Node)) {
	if pre != nil {
		pre(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		forEachNode(c, pre, post)
	}
	if post != nil {
		post(n)
	}
}

func outline(stack []string, n *html.Node) {
	if n.Type == html.ElementNode {
		stack = append(stack, n.Data)
		fmt.Println(stack)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		outline(stack, c)
	}
}

// 递归同类element 的数量
func CountElements(counts map[string]int, n *html.Node) map[string]int {
	if n == nil {
		return counts
	}
	if n.Type == html.ElementNode {
		counts[n.Data]++
	}
	// 递归调用
	counts = CountElements(counts, n.FirstChild)
	counts = CountElements(counts, n.NextSibling)
	return counts
}

// 打印同类element 的数量
func PrintElementCounts(counts map[string]int) {
	for k, v := range counts {
		fmt.Printf("%s: %d\n", k, v)
	}
}

// 编写函数输出所有text结点的内容。注意不要访问<script>和<style>元素，因为这些元素对浏览者是不可见的。
func CollectVisibleText(texts []string, n *html.Node, skip bool) []string {
	if n == nil {
		return texts
	}
	if n.Type == html.ElementNode && n.Data == "script" || n.Type == html.ElementNode && n.Data == "style" {
		skip = true
	} else if !skip && n.Type == html.TextNode {
		texts = append(texts, n.Data)
	}
	// 递归调用
	// 子节点要传递 skip ， 同为不可见文本
	texts = CollectVisibleText(texts, n.FirstChild, skip)
	// 兄弟节点不递 skip ， 不影响
	texts = CollectVisibleText(texts, n.NextSibling, false)
	return texts
}

func PrintVisibleText(texts []string) {
	fmt.Println("=== Visible Text Content ===")
	for _, text := range texts {
		fmt.Print(text)
	}
}
