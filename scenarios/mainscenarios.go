package scenarios

import (
	"context"
	"strconv"
	"time"

	"go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Each iteration executes a single workflow with a number of child workflows and/or activities. " +
			"Additional options: children-per-workflow, activities-per-workflow.",
		Executor: loadgen.KitchenSinkExecutor{
			TestInput: &kitchensink.TestInput{
				WorkflowInput: &kitchensink.WorkflowInput{
					InitialActions: []*kitchensink.ActionSet{},
				},
			},

			PrepareTestInput: func(ctx context.Context, opts loadgen.ScenarioInfo, params *kitchensink.TestInput) error {
				// Для каждого дочернего рабочего процесса готовим набор действий
				delayAction := kitchensink.ExecuteActivityAction_Delay{
					Delay: &durationpb.Duration{
						Seconds: 0,
						Nanos:   200000000, // 200 миллисекунд в наносекундах
					},
				}

				d := durationpb.New(time.Millisecond)
				d.Seconds = 1
				for i := 0; i < opts.ScenarioOptionInt("children-per-workflow", 10); i++ {
					// Создаем ActionSet для дочернего рабочего процесса
					childActions := make([]*kitchensink.Action, 0, opts.ScenarioOptionInt("activities-per-workflow", 10))
					for j := 0; j < opts.ScenarioOptionInt("activities-per-workflow", 10); j++ {
						childActions = append(childActions, &kitchensink.Action{
							Variant: &kitchensink.Action_ExecActivity{
								ExecActivity: &kitchensink.ExecuteActivityAction{
									ActivityType:        &delayAction,
									StartToCloseTimeout: &durationpb.Duration{Seconds: 5},
								},
							},
						})
					}

					childWorkflowInput := &kitchensink.WorkflowInput{
						InitialActions: []*kitchensink.ActionSet{{
							Actions:    childActions,
							Concurrent: true, // Если нужно выполнить активности параллельно
						}},
					}
					childWorkflowInput.InitialActions = append(childWorkflowInput.InitialActions,
						&kitchensink.ActionSet{
							Actions: []*kitchensink.Action{
								{
									Variant: &kitchensink.Action_ReturnResult{
										ReturnResult: &kitchensink.ReturnResultAction{
											ReturnThis: &common.Payload{},
										},
									},
								},
							},
						},
					)
					childInput, err := converter.GetDefaultDataConverter().ToPayload(childWorkflowInput)
					if err != nil {
						return err
					}

					// Добавляем дочерний рабочий процесс с подготовленным набором действий
					params.WorkflowInput.InitialActions = append(params.WorkflowInput.InitialActions, &kitchensink.ActionSet{
						Actions: []*kitchensink.Action{{
							Variant: &kitchensink.Action_ExecChildWorkflow{
								ExecChildWorkflow: &kitchensink.ExecuteChildWorkflowAction{
									WorkflowId:   opts.RunID + "-child-wf-" + strconv.Itoa(i),
									WorkflowType: "kitchenSink-child",
									Input:        []*common.Payload{childInput},
								},
							},
						}},
					})

				}

				params.WorkflowInput.InitialActions = append(params.WorkflowInput.InitialActions,
					&kitchensink.ActionSet{
						Actions: []*kitchensink.Action{
							{
								Variant: &kitchensink.Action_ReturnResult{
									ReturnResult: &kitchensink.ReturnResultAction{
										ReturnThis: &common.Payload{},
									},
								},
							},
						},
					},
				)

				return nil
			},
		},
	})
}
