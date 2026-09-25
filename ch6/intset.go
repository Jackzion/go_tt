package geometry

import (
	"bytes"
	"fmt"
)

// An IntSet is a set of small non-negative integers.
type IntSet struct {
	words []uint64
}

// Has reports whether the set contains the non-negative value x.
func (s *IntSet) Has(x int) bool {
	word , bit := x/64 , uint(x%64)
	return word < len(s.words) && s.words[word]&(1<<bit) !=0
}

// Add adds the non-negative value x to the set
func (s *IntSet) Add(x int ) {
	word , bit := x/64 , uint(x%64)
	for word >= len(s.words){
		s.words = append(s.words , 0)
	}
	s.words[word] |= 1 << bit
}

// Remove removes the non-negative value x from the set
func (s *IntSet) Remove(x int ) {
	word , bit := x/64 , uint(x%64)
	if(word < len(s.words)){
		s.words[word] &= ^(1 << bit)
	}
}

// Copy returns a copy of the set
func (s *IntSet) Copy() *IntSet {
	var newSet IntSet
	newSet.words = append(newSet.words , s.words...)
	return &newSet
}

// AddAll adds a list of values to the set
func (s *IntSet) AddAll(values ...int) {
	for _, v := range values {
		s.Add(v)
	}
}

// Clear removes all elements from the set
func (s *IntSet) Clear() {
	s.words = nil
}

// UnionWith sets s to the union of s and t
func (s *IntSet) UnionWith(t *IntSet){
	for i , tword := range t.words {
		if i < len(s.words){
			s.words[i] |= tword
		}else{
			s.words = append(s.words , tword)
		}
	}
}

// IntersectWith sets s to the intersection of s and t
func (s *IntSet) IntersectWith(t *IntSet) {

	for i , _ := range s.words {
		if i >= len(t.words){
			s.words[i] = 0
		}else{
			s.words[i] &= t.words[i]
		}
	}
}

func (s *IntSet) DifferenceWith(t *IntSet) {
	for i , _ := range s.words {
		if i >= len(t.words){
			continue 
		}else{
			s.words[i] &= ^t.words[i]
		}
	}
}

// String returns the set as a string of the form "{1 2 3}"
func (s *IntSet) String() string {

	var buf bytes.Buffer
	buf.WriteByte('{')
	for i , word := range s.words {
		if word == 0 {
			continue
		}
		for j := 0 ; j < 64 ; j++ {
			if word&(1<<uint(j)) !=0{
				if buf.Len() > len("{") {
					buf.WriteByte(' ')
				}
				fmt.Fprintf(&buf , "%d" , 64*i+j)
			}
		}
	}
	buf.WriteByte('}')
	return buf.String()
}

// Len returns the number of elements in the set.
func (s *IntSet) Len() int {
	count := 0
	for _ , word := range s.words {
		if word ==0 {
			continue
		}
		// 不为空 ， 遍历 word.bits
		for j := 0 ; j < 64 ; j++ {
			if word&(1<<uint(j)) !=0{
				count++
			}
		}
	}
	return count
}



func TestIntSet() {

	var x, y IntSet
	x.Add(1)
	x.Add(144)
	x.Add(9)
	fmt.Println(x.String()) // "{1 9 144}"

	y.Add(9)
	y.Add(42)
	fmt.Println(y.String()) // "{9 42}"

	x.DifferenceWith(&y)
	println(x.String()) // "{1 144}"

	// x.UnionWith(&y)
	// fmt.Println(x.String()) // "{1 9 42 144}"
	// fmt.Println(x.Has(9), x.Has(123)) // "true false"

}