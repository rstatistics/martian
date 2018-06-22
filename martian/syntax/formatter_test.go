//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Martian formatter tests.
//

package syntax

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestFormatValueExpression(t *testing.T) {
	ve := ValExp{
		Node:  AstNode{SourceLoc{0, new(SourceFile)}, nil, nil},
		Kind:  "float",
		Value: 0,
	}

	//
	// Format float ValExps.
	//
	var buff strings.Builder
	ve.Kind = "float"

	ve.Value = 10.0
	ve.format(&buff, "")
	assert.Equal(t, buff.String(), "10", "Preserve single zero after decimal.")
	buff.Reset()

	ve.Value = 10.05
	ve.format(&buff, "")
	assert.Equal(t, buff.String(), "10.05", "Do not strip numbers ending in non-zero digit.")
	buff.Reset()

	ve.Value = 10.050
	ve.format(&buff, "")
	assert.Equal(t, buff.String(), "10.05", "Strip single trailing zero.")
	buff.Reset()

	ve.Value = 10.050000000
	ve.format(&buff, "")
	assert.Equal(t, buff.String(), "10.05", "Strip multiple trailing zeroes.")
	buff.Reset()

	ve.Value = 0.0000000005
	ve.format(&buff, "")
	assert.Equal(t, buff.String(), "5e-10", "Handle exponential notation.")
	buff.Reset()

	ve.Value = 0.0005
	ve.format(&buff, "")
	assert.Equal(t, buff.String(), "0.0005", "Handle low decimal floats.")
	buff.Reset()

	//
	// Format int ValExps.
	//
	ve.Kind = "int"

	ve.Value = 0
	ve.format(&buff, "")
	assert.Equal(t, buff.String(), "0", "Format zero integer.")
	buff.Reset()

	ve.Value = 10
	ve.format(&buff, "")
	assert.Equal(t, buff.String(), "10", "Format non-zero integer.")
	buff.Reset()

	ve.Value = 1000000
	ve.format(&buff, "")
	assert.Equal(t, buff.String(), "1000000", "Preserve integer trailing zeroes.")
	buff.Reset()

	//
	// Format string ValExps.
	//
	ve.Kind = "string"

	ve.Value = "blah"
	ve.format(&buff, "")
	assert.Equal(t, buff.String(), "\"blah\"", "Double quote a string.")
	buff.Reset()

	ve.Value = "\"blah\""
	ve.format(&buff, "")
	assert.Equal(t, buff.String(), "\"\"blah\"\"", "Double quote a double-quoted string.")
	buff.Reset()

	//
	// Format nil ValExps.
	//
	ve.Value = nil
	ve.format(&buff, "")
	assert.Equal(t, buff.String(), "null", "Nil value is 'null'.")
}

func TestFormatCommentedSrc(t *testing.T) {
	src := `# A super-simple test pipeline with forks.

# I am good at documenting my code with useful headers.

# Get my other stuff.
@include "my_special_stuff.mro"

# Files storing json.
filetype json;
filetype txt;

# Adds a key to the json in a file.
stage ADD_KEY1(
    # The key to add
    in  string key,
    # The value to add for this key.
    in  string value,
    # The file to read the initial dictionary from.
    in  json   start,
    # A file to check.  If the file exists, parse its content as a signal
    # for the job to send to itself.
    in  string failfile,
    # The output file.
    out json   result,
    # The source file.
    src py     "stages/add_key",
)

# Some more explanation of what I'm doing could go here.

# Adds a second key to the json in a file.
stage ADD_KEY2(
    in  string key       "The key to add",
    in  string value     "The value to set the key to",
    in  json   start,
    in  string failfile  "The file to check to force failure.",
    out json   result,
    src py     "stages/add_key",
)

# Adds a third key to the json in a file.
stage ADD_KEY3(
    in  string key,
    in  string value,
    in  json   start,
    in  string failfile,
    out json   result,
    out bool   disable_example,
    src py     "stages/add_key",
) retain (
    resut,
)

stage SUM_SQUARES(
    in  float[] values,
    out float   sum,
    src comp    "bin/sum_squares mode_arg",
) split (
    in  float   value,
    out float   square,
) using (
    # For some reason this uses lots of memory.
    mem_gb   = 4,
    # This doesn't generate files anyway.
    volatile = strict,
)

# Takes two files containing json dictionaries and merges them.
stage MERGE_JSON(
    in  json json1,
    in  json json2,
    out json result,
    src py   "stages/merge_json",
)

stage MAP_EXAMPLE(
    in  map foo,
    src py  "stages/merge_json",
) using (
    mem_gb   = 2,
    # This stage always uses 4 threads!
    threads  = 4,
    volatile = strict,
)

# Adds some keys to some json files and then merges them.
pipeline AWESOME(
    in  string key1,
    in  string value1,
    in  string key2,
    in  string value2,
    out json   outfile,
)
{
    call ADD_KEY1(
        key      = self.key1,
        value    = self.value1,
        failfile = "fail1",
        start    = null,
    ) using (
        local = true,
    )

    call ADD_KEY2(
        key      = self.key2,
        value    = self.value2,
        failfile = "fail2",
        start    = ADD_KEY1.result,
    )

    call ADD_KEY3(
        key      = "3",
        value    = "three",
        failfile = "fail3",
        start    = ADD_KEY2.result,
    )

    call ADD_KEY1 as ADD_KEY4(
        key      = "4",
        value    = sweep(
            "four",
            "feir",
        ),
        failfile = "fail4",
        start    = ADD_KEY2.result,
    )

    call MAP_EXAMPLE(
        foo = {
            "bar": "baz",
            "bing": null,
            "blarg": {
                "n": 2,
            },
        },
    ) using (
        # ADD_KEY3 can disable this stage.
        disabled = ADD_KEY3.disable_example,
        local    = true,
        # This shouldn't be volatile because reasons.
        volatile = false,
    )

    call ADD_KEY5(
        key   = "5",
        value = ["five"],
    ) using (
        volatile = true,
    )

    call ADD_KEY6(
        key   = "6",
        value = [
            "six",
            "seven",
        ],
    )

    call MERGE_JSON(
        json1 = ADD_KEY3.result,
        json2 = ADD_KEY4.result,
    )

    call MERGE_JSON2(
        input = [ADD_KEY3.result],
    )

    call MERGE_JSON3(
        input = [
            ADD_KEY3.result,
            ADD_KEY4.result,
        ],
    )

    call MERGE_JSON4(
        input = [
            "four",
            ADD_KEY4.result,
        ],
    )

    call MERGE_JSON5(
        input = [],
    )

    return (
        outfile = MERGE_JSON.result,
    )

    retain (
        ADD_KEY1.result,
    )
}

# Calls the pipelines, sweeping over two forks.
call AWESOME(
    key1   = "1",
    value1 = "one",
    key2   = "2",
    value2 = sweep(
        "two",
        "deux",
    ),
)
`
	if formatted, err := Format(src, "test"); err != nil {
		t.Errorf("Format error: %v", err)
	} else if formatted != src {
		diffLines(src, formatted, t)
	}
}

func TestFormatTopoSort(t *testing.T) {
	const src = `pipeline PIPELINE(
    in  int input,
    out int output1,
    out int output2,
)
{
    call STAGE_3(
        in1 = self.input,
        in2 = STAGE_2.output,
    )

    call STAGE as STAGE_1(
        input = self.input,
    )

    call STAGE as STAGE_2(
        input = self.input,
    )

    call STAGE as STAGE_4(
        input = STAGE_2.output,
    )

    call STAGE_3 as STAGE_5(
        in1 = self.input,
        in2 = STAGE_2.output,
    )

    return (
        output1 = STAGE_3.output,
        output2 = STAGE_5.output,
    )
}
`
	const expected = `pipeline PIPELINE(
    in  int input,
    out int output1,
    out int output2,
)
{
    call STAGE as STAGE_1(
        input = self.input,
    )

    call STAGE as STAGE_2(
        input = self.input,
    )

    call STAGE_3(
        in1 = self.input,
        in2 = STAGE_2.output,
    )

    call STAGE as STAGE_4(
        input = STAGE_2.output,
    )

    call STAGE_3 as STAGE_5(
        in1 = self.input,
        in2 = STAGE_2.output,
    )

    return (
        output1 = STAGE_3.output,
        output2 = STAGE_5.output,
    )
}
`
	if formatted, err := Format(src, "test"); err != nil {
		t.Errorf("Format error: %v", err)
	} else if formatted != expected {
		diffLines(expected, formatted, t)
	}
}

func diffLines(src, formatted string, t *testing.T) {
	src_lines := strings.Split(src, "\n")
	formatted_lines := strings.Split(formatted, "\n")
	offset := 0
	for i, line := range src_lines {
		pad := ""
		if len(line) < 30 {
			pad = strings.Repeat(" ", 30-len(line))
		}
		if len(formatted_lines) > i+offset {
			if line == formatted_lines[i+offset] {
				t.Logf("%3d: %s %s= %s", i, line, pad, formatted_lines[i+offset])
			} else if strings.TrimSpace(line) == strings.TrimSpace(formatted_lines[i+offset]) {
				t.Errorf("%3d: %s %s| %s", i, line, pad, formatted_lines[i+offset])
			} else {
				forwardOffset := 0
				for moreOffset, fline := range formatted_lines[i+offset:] {
					if strings.TrimSpace(fline) == strings.TrimSpace(line) {
						forwardOffset = moreOffset
						break
					} else if moreOffset > 20 {
						break
					}
				}
				backwardOffset := 0
				for moreOffset, uline := range src_lines[i:] {
					if strings.TrimSpace(uline) == strings.TrimSpace(formatted_lines[i+offset]) {
						backwardOffset = moreOffset
						break
					} else if moreOffset > 20 {
						break
					}
				}
				//				t.Logf("offsets %d and %d", forwardOffset, backwardOffset)
				if forwardOffset == 0 && backwardOffset == 0 {
					t.Errorf("%3d: %s %s| %s", i, line, pad, formatted_lines[i+offset])
				} else if (forwardOffset == 0 && backwardOffset != 0) ||
					(backwardOffset > forwardOffset) {
					t.Errorf("%3d: %s %s<", i, line, pad)
					offset--
				} else {
					for j := 0; j < forwardOffset; j++ {
						t.Errorf("%s > %s", strings.Repeat(" ", 35), formatted_lines[i+j+offset])
					}
					offset += forwardOffset
					if line == formatted_lines[i+offset] {
						t.Logf("%3d: %s %s= %s", i, line, pad, formatted_lines[i+offset])
					} else {
						t.Errorf("%3d: %s %s| %s", i, line, pad, formatted_lines[i+offset])
					}
				}
			}
		} else {
			t.Errorf("%3d: %s %s<", i, line, pad)
		}
	}
	if len(formatted_lines) > len(src_lines)+offset {
		for _, line := range formatted_lines[len(src_lines)+offset:] {
			t.Errorf("%s > %s", strings.Repeat(" ", 35), line)
		}
	}
}
