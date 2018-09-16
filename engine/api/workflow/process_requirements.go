package workflow

import (
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/interpolate"
)

// getNodeJobRunRequirements returns requirements list interpolated, and true or false if at least
// one requirement is of type "Service"
func getNodeJobRunRequirements(db gorp.SqlExecutor, j sdk.Job, run *sdk.WorkflowNodeRun) (sdk.RequirementList, bool, *sdk.MultiError) {
	requirements := sdk.RequirementList{}
	tmp := map[string]string{}
	errm := &sdk.MultiError{}

	var containsService bool
	for _, v := range run.BuildParameters {
		tmp[v.Name] = v.Value
	}

	for _, v := range j.Action.Requirements {
		name, errName := interpolate.Do(v.Name, tmp)
		if errName != nil {
			errm.Append(errName)
			continue
		}
		value, errValue := interpolate.Do(v.Value, tmp)
		if errValue != nil {
			errm.Append(errValue)
			continue
		}
		sdk.AddRequirement(&requirements, v.ID, name, v.Type, value)
		if v.Type == sdk.ServiceRequirement {
			containsService = true
		}
	}

	if errm.IsEmpty() {
		return requirements, containsService, nil
	}
	return requirements, containsService, errm
}

func prepareRequirementsToNodeJobRunParameters(reqs sdk.RequirementList) []sdk.Parameter {
	params := []sdk.Parameter{}
	for _, r := range reqs {
		if r.Type == sdk.ServiceRequirement {
			k := fmt.Sprintf("job.requirement.%s.%s", strings.ToLower(r.Type), strings.ToLower(r.Name))
			values := strings.Split(r.Value, " ")
			if len(values) > 1 {
				sdk.AddParameter(&params, k+".image", sdk.StringParameter, values[0])
				sdk.AddParameter(&params, k+".options", sdk.StringParameter, strings.Join(values[1:], " "))
			}
		}
		k := fmt.Sprintf("job.requirement.%s.%s", strings.ToLower(r.Type), strings.ToLower(r.Name))
		sdk.AddParameter(&params, k, sdk.StringParameter, r.Value)
	}
	return params
}
