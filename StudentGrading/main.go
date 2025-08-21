package main

import (
	grading "StudentGrading/Utils"
)

func main() {
	originalMarks := [10]int{93, 82, 47, 88, 55, 35, 67, 39, 100, 76}

	finalMarks, _ := grading.ReExam(originalMarks[:])

	grading.Results(finalMarks, originalMarks[:])
}
