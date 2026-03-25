package cmd

import (
	"context"
	"net/http"

	"orquesta/deployapp"
)

type apiDeployDockerRemoteRequest struct {
	Spec         deployapp.DockerRemoteSpec `json:"spec"`
	DryRun       bool                       `json:"dry_run"`
	AutoRollback bool                       `json:"auto_rollback"`
}

type apiDeployDockerRemotePlanResponse struct {
	Plan *deployapp.DockerRemotePlan `json:"plan"`
}

type apiDeployDockerRemoteExecuteResponse struct {
	Plan   *deployapp.DockerRemotePlan `json:"plan"`
	Result *deployapp.ExecuteResult    `json:"result"`
}

func apiHandlerDeployDockerRemotePlan(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiDeployDockerRemoteRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	plan, err := deployapp.NewService().Plan(req.Spec)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiDeployDockerRemotePlanResponse{Plan: plan})
}

func apiHandlerDeployDockerRemoteExecute(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiDeployDockerRemoteRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	service := deployapp.NewService()
	plan, err := service.Plan(req.Spec)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	result, err := service.Execute(context.Background(), plan, deployapp.ExecuteOptions{
		DryRun:       req.DryRun,
		AutoRollback: req.AutoRollback,
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiDeployDockerRemoteExecuteResponse{
		Plan:   plan,
		Result: result,
	})
}
