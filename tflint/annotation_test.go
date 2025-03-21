package tflint

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func Test_NewAnnotations(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		src      string
		want     Annotations
		diags    string
	}{
		{
			name:     "annotation starting with #",
			filename: "resource.tf",
			src: `
resource "aws_instance" "foo" {
  # tflint-ignore: aws_instance_invalid_type
  instance_type = "t2.micro" # This is also comment
}`,
			want: Annotations{
				&LineAnnotation{
					Content: "aws_instance_invalid_type",
					Token: hclsyntax.Token{
						Type:  hclsyntax.TokenComment,
						Bytes: []byte("# tflint-ignore: aws_instance_invalid_type\n"),
						Range: hcl.Range{
							Filename: "resource.tf",
							Start:    hcl.Pos{Line: 3, Column: 3},
							End:      hcl.Pos{Line: 4, Column: 1},
						},
					},
				},
			},
		},
		{
			name:     "annotation starting with //",
			filename: "resource.tf",
			src: `
resource "aws_instance" "foo" {
  // This is also comment
  instance_type = "t2.micro" // tflint-ignore: aws_instance_invalid_type
}`,
			want: Annotations{
				&LineAnnotation{
					Content: "aws_instance_invalid_type",
					Token: hclsyntax.Token{
						Type:  hclsyntax.TokenComment,
						Bytes: []byte("// tflint-ignore: aws_instance_invalid_type\n"),
						Range: hcl.Range{
							Filename: "resource.tf",
							Start:    hcl.Pos{Line: 4, Column: 30},
							End:      hcl.Pos{Line: 5, Column: 1},
						},
					},
				},
			},
		},
		{
			name:     "annotation starting with /*",
			filename: "resource.tf",
			src: `
resource "aws_instance" "foo" {
  /* tflint-ignore: aws_instance_invalid_type */
  instance_type = "t2.micro" /* This is also comment */
}`,
			want: Annotations{
				&LineAnnotation{
					Content: "aws_instance_invalid_type",
					Token: hclsyntax.Token{
						Type:  hclsyntax.TokenComment,
						Bytes: []byte("/* tflint-ignore: aws_instance_invalid_type */"),
						Range: hcl.Range{
							Filename: "resource.tf",
							Start:    hcl.Pos{Line: 3, Column: 3},
							End:      hcl.Pos{Line: 3, Column: 49},
						},
					},
				},
			},
		},
		{
			name:     "ignoring multiple rules",
			filename: "resource.tf",
			src: `
resource "aws_instance" "foo" {
  /* tflint-ignore: aws_instance_invalid_type, terraform_deprecated_syntax */
  instance_type = "t2.micro"
}`,
			want: Annotations{
				&LineAnnotation{
					Content: "aws_instance_invalid_type, terraform_deprecated_syntax",
					Token: hclsyntax.Token{
						Type:  hclsyntax.TokenComment,
						Bytes: []byte("/* tflint-ignore: aws_instance_invalid_type, terraform_deprecated_syntax */"),
						Range: hcl.Range{
							Filename: "resource.tf",
							Start:    hcl.Pos{Line: 3, Column: 3},
							End:      hcl.Pos{Line: 3, Column: 78},
						},
					},
				},
			},
		},
		{
			name:     "with reason starting with //",
			filename: "resource.tf",
			src: `
resource "aws_instance" "foo" {
  instance_type = "t2.micro" // tflint-ignore: aws_instance_invalid_type // With reason
}`,
			want: Annotations{
				&LineAnnotation{
					Content: "aws_instance_invalid_type",
					Token: hclsyntax.Token{
						Type:  hclsyntax.TokenComment,
						Bytes: []byte("// tflint-ignore: aws_instance_invalid_type // With reason\n"),
						Range: hcl.Range{
							Filename: "resource.tf",
							Start:    hcl.Pos{Line: 3, Column: 30},
							End:      hcl.Pos{Line: 4, Column: 1},
						},
					},
				},
			},
		},
		{
			name:     "with reason starting with #",
			filename: "resource.tf",
			src: `
resource "aws_instance" "foo" {
  # tflint-ignore: aws_instance_invalid_type # With reason
  instance_type = "t2.micro"
}`,
			want: Annotations{
				&LineAnnotation{
					Content: "aws_instance_invalid_type",
					Token: hclsyntax.Token{
						Type:  hclsyntax.TokenComment,
						Bytes: []byte("# tflint-ignore: aws_instance_invalid_type # With reason\n"),
						Range: hcl.Range{
							Filename: "resource.tf",
							Start:    hcl.Pos{Line: 3, Column: 3},
							End:      hcl.Pos{Line: 4, Column: 1},
						},
					},
				},
			},
		},
		{
			name:     "tflint-ignore-file annotation",
			filename: "resource.tf",
			src: `# tflint-ignore-file: aws_instance_invalid_type
resource "aws_instance" "foo" {
  instance_type = "t2.micro"
}`,
			want: Annotations{
				&FileAnnotation{
					Content: "aws_instance_invalid_type",
					Token: hclsyntax.Token{
						Type:  hclsyntax.TokenComment,
						Bytes: []byte("# tflint-ignore-file: aws_instance_invalid_type\n"),
						Range: hcl.Range{
							Filename: "resource.tf",
							Start:    hcl.Pos{Line: 1, Column: 1},
							End:      hcl.Pos{Line: 2, Column: 1},
						},
					},
				},
			},
		},
		{
			name:     "tflint-ignore-file annotation outside the first line",
			filename: "resource.tf",
			src: `
resource "aws_instance" "foo" {
  # tflint-ignore-file: aws_instance_invalid_type
  instance_type = "t2.micro"
}`,
			want:  Annotations{},
			diags: "resource.tf:3,3-4,1: tflint-ignore-file annotation must be written at the top of file; tflint-ignore-file annotation is written at line 3, column 3",
		},
		{
			name:     "tflint-ignore-file annotation outside the first column",
			filename: "resource.tf",
			src: `resource "aws_instance" "foo" { # tflint-ignore-file: aws_instance_invalid_type
  instance_type = "t2.micro"
}`,
			want:  Annotations{},
			diags: "resource.tf:1,33-2,1: tflint-ignore-file annotation must be written at the top of file; tflint-ignore-file annotation is written at line 1, column 33",
		},
		{
			name:     "tflint-ignore-file in JSON comment property",
			filename: "resource.tf.json",
			src: `{
  "//": "tflint-ignore-file: aws_instance_invalid_type",
  "resource": {
    "aws_instance": {
      "foo": {
        "instance_type": "t2.micro"
      }
    }
  }
}`,
			want: Annotations{
				&FileAnnotation{
					Content: "aws_instance_invalid_type",
					Token: hclsyntax.Token{
						Range: hcl.Range{
							Filename: "resource.tf.json",
						},
					},
				},
			},
		},
		{
			name:     "tflint-ignore-file with multiple rules in JSON comment property and following comment",
			filename: "resource.tf.json",
			src: `{
  "//": "tflint-ignore-file: aws_instance_invalid_type, terraform_deprecated_syntax # this is an extra comment",
  "resource": {
	"aws_instance": {
	  "foo": {
		"instance_type": "t2.micro"
	  }
	}
  }
}`,
			want: Annotations{
				&FileAnnotation{
					Content: "aws_instance_invalid_type, terraform_deprecated_syntax",
					Token: hclsyntax.Token{
						Range: hcl.Range{
							Filename: "resource.tf.json",
						},
					},
				},
			},
		},
		{
			name:     "no errors if JSON comment property is not the expected structure",
			filename: "resource.tf.json",
			src: `{
  "//": {"foo": "bar"},
  "resource": {
	"aws_instance": {
	  "foo": {
		"instance_type": "t2.micro"
	  }
	}
  }
}`,
			want: Annotations{},
		},
		{
			name:     "tflint-ignore-file annotation outside the first column of the JSON comment property",
			filename: "resource.tf.json",
			src: `{
  "//": "blah blah # tflint-ignore-file: aws_instance_invalid_type",
  "resource": {
	"aws_instance": {
	  "foo": {
		"instance_type": "t2.micro"
	  }
	}
  }
}`,
			want:  Annotations{},
			diags: "resource.tf.json:0,0-0: tflint-ignore-file annotation must appear at the beginning of the JSON comment property value; tflint-ignore-file annotation is written at index 12 of the comment property value",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parser := hclparse.NewParser()
			var (
				file  *hcl.File
				diags hcl.Diagnostics
			)
			switch {
			case strings.HasSuffix(test.filename, ".json"):
				file, diags = parser.ParseJSON([]byte(test.src), test.filename)
			default:
				file, diags = parser.ParseHCL([]byte(test.src), test.filename)
			}
			if diags.HasErrors() {
				t.Fatal(diags)
			}

			got, diags := NewAnnotations(test.filename, file)
			if diags.HasErrors() || test.diags != "" {
				if diags.Error() != test.diags {
					t.Errorf("want=%s, got=%s", test.diags, diags.Error())
				}
			}

			opts := cmpopts.IgnoreFields(hcl.Pos{}, "Byte")
			if diff := cmp.Diff(test.want, got, opts); diff != "" {
				t.Errorf("diff: %s", diff)
			}
		})
	}
}

func TestLineAnnotation_IsAffected(t *testing.T) {
	issue := &Issue{
		Rule:    &testRule{},
		Message: "Test rule",
		Range: hcl.Range{
			Filename: "test.tf",
			Start:    hcl.Pos{Line: 2},
		},
	}

	tests := []struct {
		Name       string
		Annotation *LineAnnotation
		Expected   bool
	}{
		{
			Name: "affected (same line)",
			Annotation: &LineAnnotation{
				Content: "test_rule",
				Token: hclsyntax.Token{
					Type: hclsyntax.TokenComment,
					Range: hcl.Range{
						Filename: "test.tf",
						Start:    hcl.Pos{Line: 2},
					},
				},
			},
			Expected: true,
		},
		{
			Name: "affected (above line)",
			Annotation: &LineAnnotation{
				Content: "test_rule",
				Token: hclsyntax.Token{
					Type: hclsyntax.TokenComment,
					Range: hcl.Range{
						Filename: "test.tf",
						Start:    hcl.Pos{Line: 1},
					},
				},
			},
			Expected: true,
		},
		{
			Name: "affected (multiple rules)",
			Annotation: &LineAnnotation{
				Content: "other_rule, test_rule",
				Token: hclsyntax.Token{
					Type: hclsyntax.TokenComment,
					Range: hcl.Range{
						Filename: "test.tf",
						Start:    hcl.Pos{Line: 2},
					},
				},
			},
			Expected: true,
		},
		{
			Name: "not affected (multiple rules)",
			Annotation: &LineAnnotation{
				Content: "other_rule_a, other_rule_b",
				Token: hclsyntax.Token{
					Type: hclsyntax.TokenComment,
					Range: hcl.Range{
						Filename: "test.tf",
						Start:    hcl.Pos{Line: 2},
					},
				},
			},
			Expected: false,
		},
		{
			Name: "not affected (under line)",
			Annotation: &LineAnnotation{
				Content: "test_rule",
				Token: hclsyntax.Token{
					Type: hclsyntax.TokenComment,
					Range: hcl.Range{
						Filename: "test.tf",
						Start:    hcl.Pos{Line: 3},
					},
				},
			},
			Expected: false,
		},
		{
			Name: "not affected (another filename)",
			Annotation: &LineAnnotation{
				Content: "test_rule",
				Token: hclsyntax.Token{
					Type: hclsyntax.TokenComment,
					Range: hcl.Range{
						Filename: "test2.tf",
						Start:    hcl.Pos{Line: 2},
					},
				},
			},
			Expected: false,
		},
		{
			Name: "not affected (another rule)",
			Annotation: &LineAnnotation{
				Content: "test_another_rule",
				Token: hclsyntax.Token{
					Type: hclsyntax.TokenComment,
					Range: hcl.Range{
						Filename: "test.tf",
						Start:    hcl.Pos{Line: 2},
					},
				},
			},
			Expected: false,
		},
		{
			Name: "affected (all)",
			Annotation: &LineAnnotation{
				Content: "all",
				Token: hclsyntax.Token{
					Type: hclsyntax.TokenComment,
					Range: hcl.Range{
						Filename: "test.tf",
						Start:    hcl.Pos{Line: 2},
					},
				},
			},
			Expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			got := test.Annotation.IsAffected(issue)
			if got != test.Expected {
				t.Fatalf("want=%t, got=%t", test.Expected, got)
			}
		})
	}
}

func TestFileAnnotation_IsAffected(t *testing.T) {
	issue := &Issue{
		Rule:    &testRule{},
		Message: "Test rule",
		Range: hcl.Range{
			Filename: "test.tf",
			Start:    hcl.Pos{Line: 2},
		},
	}

	tests := []struct {
		Name       string
		Annotation *FileAnnotation
		Expected   bool
	}{
		{
			Name: "affected",
			Annotation: &FileAnnotation{
				Content: "test_rule",
				Token: hclsyntax.Token{
					Type: hclsyntax.TokenComment,
					Range: hcl.Range{
						Filename: "test.tf",
					},
				},
			},
			Expected: true,
		},
		{
			Name: "not affected (another filename)",
			Annotation: &FileAnnotation{
				Content: "test_rule",
				Token: hclsyntax.Token{
					Type: hclsyntax.TokenComment,
					Range: hcl.Range{
						Filename: "test2.tf",
					},
				},
			},
			Expected: false,
		},
		{
			Name: "affected (multiple rules)",
			Annotation: &FileAnnotation{
				Content: "other_rule, test_rule",
				Token: hclsyntax.Token{
					Type: hclsyntax.TokenComment,
					Range: hcl.Range{
						Filename: "test.tf",
					},
				},
			},
			Expected: true,
		},
		{
			Name: "not affected (multiple rules)",
			Annotation: &FileAnnotation{
				Content: "other_rule_a, other_rule_b",
				Token: hclsyntax.Token{
					Type: hclsyntax.TokenComment,
					Range: hcl.Range{
						Filename: "test.tf",
					},
				},
			},
			Expected: false,
		},
		{
			Name: "not affected (another rule)",
			Annotation: &FileAnnotation{
				Content: "test_another_rule",
				Token: hclsyntax.Token{
					Type: hclsyntax.TokenComment,
					Range: hcl.Range{
						Filename: "test.tf",
					},
				},
			},
			Expected: false,
		},
		{
			Name: "affected (all)",
			Annotation: &FileAnnotation{
				Content: "all",
				Token: hclsyntax.Token{
					Type: hclsyntax.TokenComment,
					Range: hcl.Range{
						Filename: "test.tf",
					},
				},
			},
			Expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			got := test.Annotation.IsAffected(issue)
			if got != test.Expected {
				t.Fatalf("want=%t, got=%t", test.Expected, got)
			}
		})
	}
}
