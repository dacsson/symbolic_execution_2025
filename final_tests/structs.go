package final_tests

import (
	"errors"
)

type ObjectWithPrimitivesClass struct {
	ValueByDefault int
	x, y           int
	ShortValue     int16
	Weight         float64
}

func NewObjectWithPrimitivesClass() *ObjectWithPrimitivesClass {
	return &ObjectWithPrimitivesClass{ValueByDefault: 5}
}

func Max(fst, snd *ObjectWithPrimitivesClass) *ObjectWithPrimitivesClass {
	if fst.x > snd.x && fst.y > snd.y {
		return fst
	} else if fst.x < snd.x && fst.y < snd.y {
		return snd
	}
	return fst
}

func Example(value *ObjectWithPrimitivesClass) *ObjectWithPrimitivesClass {
	if value.x == 1 {
		return value
	}
	value.x = 1
	return value
}

func CreateObject(a, b int, objectExample *ObjectWithPrimitivesClass) (*ObjectWithPrimitivesClass, error) {
	object := NewObjectWithPrimitivesClass()
	object.x = a + 5
	object.y = b + 6
	object.Weight = objectExample.Weight
	if object.Weight < 0 {
		return nil, errors.New("IllegalArgumentException: weight < 0")
	}
	return object, nil
}

func Memory(objectExample *ObjectWithPrimitivesClass, value int) *ObjectWithPrimitivesClass {
	if value > 0 {
		objectExample.x = 1
		objectExample.y = 2
		objectExample.Weight = 1.2
	} else {
		objectExample.x = -1
		objectExample.y = -2
		objectExample.Weight = -1.2
	}
	return objectExample
}

func CompareTwoNullObjects(value int) int {
	fst := NewObjectWithPrimitivesClass()
	snd := NewObjectWithPrimitivesClass()

	fst.x = value + 1
	snd.x = value + 2

	if fst.x == value+1 {
		fst = nil
	}
	if snd.x == value+2 {
		snd = nil
	}

	if fst == snd {
		return 1
	}
	panic("RuntimeException")
}

type SimpleDataClass struct {
	a int
	b int
}

type ObjectWithRefFieldClass struct {
	x, y       int
	Weight     float64
	RefField   *SimpleDataClass
	ArrayField []int
}

func WriteToRefTypeField(objectExample *ObjectWithRefFieldClass, value int) (*ObjectWithRefFieldClass, error) {
	if value != 42 {
		return nil, errors.New("IllegalArgumentException: value must be 42")
	} else if objectExample.RefField != nil {
		return nil, errors.New("IllegalArgumentException: refField must be nil")
	}

	simpleDataClass := &SimpleDataClass{
		a: value,
		b: value * 2,
	}
	objectExample.RefField = simpleDataClass
	return objectExample, nil
}

func WriteToArrayField(objectExample *ObjectWithRefFieldClass, length int) (*ObjectWithRefFieldClass, error) {
	if length < 3 {
		return nil, errors.New("IllegalArgumentException: length must be at least 3")
	}

	array := make([]int, length)
	for i := 0; i < length; i++ {
		array[i] = i + 1
	}

	objectExample.ArrayField = array
	objectExample.ArrayField[length-1] = 100

	return objectExample, nil
}

func ReadFromArrayField(objectExample *ObjectWithRefFieldClass, value int) int {
	if len(objectExample.ArrayField) > 2 && objectExample.ArrayField[2] == value {
		return 1
	}
	return 2
}

func CompareTwoDifferentObjectsFromArguments(fst, snd *ObjectWithRefFieldClass) int {
	if fst.x > 0 && snd.x < 0 {
		if fst == snd {
			panic("RuntimeException")
		} else {
			return 1
		}
	}

	fst.x = snd.x
	fst.y = snd.y
	fst.Weight = snd.Weight

	if fst == snd {
		return 2
	}

	return 3
}

func CompareTwoObjectsWithTheSameRefField(fst, snd *ObjectWithRefFieldClass) int {
	simpleDataClass := &SimpleDataClass{}

	fst.RefField = simpleDataClass
	snd.RefField = simpleDataClass
	fst.x = snd.x
	fst.y = snd.y
	fst.Weight = snd.Weight

	if fst == snd {
		return 1
	}
	return 2
}

type RecursiveTypeClass struct {
	Next  *RecursiveTypeClass
	Value int
}

func NextValue(node *RecursiveTypeClass, value int) (*RecursiveTypeClass, error) {
	if value == 0 {
		return nil, errors.New("IllegalArgumentException: value cannot be 0")
	}
	if node.Next != nil && node.Next.Value == value {
		return node.Next, nil
	}
	return nil, nil
}

func WriteObjectField(node *RecursiveTypeClass) *RecursiveTypeClass {
	if node.Next == nil {
		node.Next = &RecursiveTypeClass{}
	}
	node.Next.Value = node.Next.Value + 1
	return node
}

type Person struct {
	Name string
	Age  int
	ID   int
}

func TestPathConstraintMutability(p Person) {
	if p.Age != 18 {
		p.Age = 18
		if p.Age != 18 {
			panic("Seems impossible")
		} else {
			println("Seems ok")
		}
	}
}
