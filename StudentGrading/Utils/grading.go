package grading

import "fmt"

func GetGrade(marks int) string {
	switch {
	case marks >= 90:
		return "Grade A"
	case marks >= 70:
		return "Grade B"
	case marks >= 50:
		return "Grade C"
	default:
		return "Fail"
	}
}

func ReExam(originalMarks []int) ([]int, []int) {
	finalMarks := make([]int, len(originalMarks))
	reExamMarks := []int{}

	for i, mark := range originalMarks {
		if mark < 50 {
			var rexMark int
			fmt.Printf("Student %d failed (marks: %d). Enter re-exam marks: ", i+1, mark)
			fmt.Scan(&rexMark)
			if rexMark > mark {
				finalMarks[i] = rexMark
			} else {
				finalMarks[i] = mark
			}
			reExamMarks = append(reExamMarks, rexMark)
		} else {
			finalMarks[i] = mark
		}
	}
	return finalMarks, reExamMarks
}

func Results(finalMarks []int, originalMarks []int) {
	totalPass := 0
	totalFail := 0

	for i, mark := range finalMarks {
		grade := GetGrade(mark)
		status := ""

		if originalMarks[i] >= 50 {
			status = "Pass (first attempt)"
		} else if mark >= 50 {
			status = "Pass (after re-exam)"
		} else {
			status = "Fail (all attempts)"
		}

		fmt.Printf("Student %d → Marks: %d     | %s     | %s\n", i+1, mark, grade, status)

		if mark >= 50 {
			totalPass++
		} else {
			totalFail++
		}
	}

	fmt.Printf("\nTotal Passed: %d\n", totalPass)
	fmt.Printf("Total Failed: %d\n", totalFail)
}
