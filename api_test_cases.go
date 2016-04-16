package dockworker

type testCase struct {
	requestBody   string
	job           Job
	resultStatus  JobStatus
	numContainers int
	numImages     int
	logs          string
}

var apiTestCases = []testCase{
	testCase{
		requestBody: `{
	  "image": "ubuntu:14.04",
	  "cmds": [
	    ["sh", "-c", "echo \"test\" > /test.txt"],
	    ["sleep", "1"],
	    ["cat", "/test.txt"]
	  ],
		"webhook_url": "%s"
	}`,
		job: Job{
			ImageName: "ubuntu:14.04",
			Cmds: []Cmd{
				[]string{"sh", "-c", "echo \"test\" > /test.txt"},
				[]string{"sleep", "1"},
				[]string{"cat", "/test.txt"},
			},
			Results: []CmdResult{0, 0, 0},
		},
		resultStatus:  JobStatusSuccessful,
		numContainers: 3,
		numImages:     3,
		logs:          "test\n",
	},
	testCase{
		requestBody: `{
	  "image": "ubuntu:14.04",
	  "cmds": [
	    ["sh", "-c", "echo \"test\" > /test.txt"],
	    ["sleep", "1"],
	    ["cat", "/notthere.txt"],
	    ["echo", "'I shouldn't run"]
	  ],
		"webhook_url": "%s"
	}`,
		job: Job{
			ImageName: "ubuntu:14.04",
			Cmds: []Cmd{
				[]string{"sh", "-c", "echo \"test\" > /test.txt"},
				[]string{"sleep", "1"},
				[]string{"cat", "/notthere.txt"},
				[]string{"echo", "'I shouldn't run"},
			},
			Results: []CmdResult{0, 0, 1},
		},
		resultStatus:  JobStatusFailed,
		numContainers: 3,
		numImages:     2,
		logs:          "cat: /notthere.txt: No such file or directory\n",
	},
	testCase{
		requestBody: `{
	  "image": "ubuntu:14.04",
	  "cmds": [
	    ["notacommand"]
	  ],
		"webhook_url": "%s"
	}`,
		job: Job{
			ImageName: "ubuntu:14.04",
			Cmds: []Cmd{
				[]string{"notacommand"},
			},
		},
		resultStatus:  JobStatusError,
		numContainers: 1,
		numImages:     0,
		logs:          "",
	},
	testCase{
		requestBody: `{
  "image": "ubuntu:14.04",
  "cmds": [
    ["sh", "-c", "echo $TEST_VAR1"],
		["sh", "-c", "echo $TEST_VAR2"]
  ],
	"env": {
		"TEST_VAR1": "test value 1",
		"TEST_VAR2": "test value 2"
	},
	"webhook_url": "%s"
}`,
		job: Job{
			ImageName: "ubuntu:14.04",
			Cmds: []Cmd{
				[]string{"sh", "-c", "echo $TEST_VAR1"},
				[]string{"sh", "-c", "echo $TEST_VAR2"},
			},
			Env: map[string]string{
				"TEST_VAR1": "test value 1",
				"TEST_VAR2": "test value 2",
			},
			Results: []CmdResult{0, 0},
		},
		resultStatus:  JobStatusSuccessful,
		numContainers: 2,
		numImages:     2,
		logs:          "test value 1\ntest value 2\n",
	},
	testCase{
		requestBody: `{
	  "image": "doesnotexist",
	  "cmds": [
	    ["echo", "$TEST_VAR1"],
			["echo", "$TEST_VAR2"]
	  ],
		"env": {
			"TEST_VAR1": "test value 1",
			"TEST_VAR2": "test value 2"
		},
		"webhook_url": "%s"
	}`,
		job: Job{
			ImageName: "doesnotexist",
			Cmds: []Cmd{
				[]string{"echo", "$TEST_VAR1"},
				[]string{"echo", "$TEST_VAR2"},
			},
			Env: map[string]string{
				"TEST_VAR1": "test value 1",
				"TEST_VAR2": "test value 2",
			},
		},
		resultStatus:  JobStatusError,
		numContainers: 0,
		numImages:     0,
	},
}
