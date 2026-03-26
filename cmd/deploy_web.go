package cmd

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"orquesta/deployapp"
)

type webDeployData struct {
	Spec         deployapp.DockerRemoteSpec
	DryRun       bool
	AutoRollback bool
	Plan         *deployapp.DockerRemotePlan
	Result       *deployapp.ExecuteResult
	Msg          string
	Err          string
}

func webHandlerDeploy(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		webHandlerDeployAccion(w, r)
		return
	}
	webRender(w, r, webTplLayout+webTplDeploy, webDeployData{
		Spec: webDeploySpecFromValues(r.URL.Query()),
		Msg:  r.URL.Query().Get("ok"),
		Err:  r.URL.Query().Get("err"),
	})
}

func webHandlerDeployAccion(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	spec := webDeploySpecFromValues(r.Form)
	data := webDeployData{
		Spec:         spec,
		DryRun:       strings.TrimSpace(r.FormValue("dry_run")) != "",
		AutoRollback: strings.TrimSpace(r.FormValue("auto_rollback")) != "",
	}
	req := apiDeployDockerRemoteRequest{
		Spec:         spec,
		DryRun:       data.DryRun,
		AutoRollback: data.AutoRollback,
	}
	if strings.TrimSpace(r.FormValue("accion")) == "ejecutar" {
		var resp apiDeployDockerRemoteExecuteResponse
		if err := webInvocarAPIJSON(http.MethodPost, "/api/deploy/docker-remoto/ejecutar", req, &resp); err != nil {
			data.Err = err.Error()
		} else {
			data.Plan = resp.Plan
			data.Result = resp.Result
			data.Msg = webTranslateRequestf(r, "deploy.flash.executed")
		}
		webRender(w, r, webTplLayout+webTplDeploy, data)
		return
	}
	var resp apiDeployDockerRemotePlanResponse
	if err := webInvocarAPIJSON(http.MethodPost, "/api/deploy/docker-remoto/plan", req, &resp); err != nil {
		data.Err = err.Error()
	} else {
		data.Plan = resp.Plan
		data.Msg = webTranslateRequestf(r, "deploy.flash.planned")
	}
	webRender(w, r, webTplLayout+webTplDeploy, data)
}

func webDeploySpecFromValues(values url.Values) deployapp.DockerRemoteSpec {
	spec := deployapp.DockerRemoteSpec{
		ProyectoSlug:   strings.TrimSpace(values.Get("proyecto")),
		Servicio:       strings.TrimSpace(values.Get("servicio")),
		SSHHost:        strings.TrimSpace(values.Get("ssh_host")),
		SSHUser:        strings.TrimSpace(values.Get("ssh_user")),
		RemoteDir:      strings.TrimSpace(values.Get("remote_dir")),
		ComposeFile:    strings.TrimSpace(values.Get("compose_file")),
		EnvFile:        strings.TrimSpace(values.Get("env_file")),
		Image:          strings.TrimSpace(values.Get("image")),
		Tag:            strings.TrimSpace(values.Get("tag")),
		RollbackTag:    strings.TrimSpace(values.Get("rollback_tag")),
		Dockerfile:     strings.TrimSpace(values.Get("dockerfile")),
		BuildContext:   strings.TrimSpace(values.Get("build_context")),
		HealthcheckCmd: strings.TrimSpace(values.Get("healthcheck_cmd")),
		Strategy:       strings.TrimSpace(values.Get("strategy")),
		Build:          strings.TrimSpace(values.Get("build")) != "",
	}
	if spec.ComposeFile == "" {
		spec.ComposeFile = "docker-compose.yml"
	}
	if spec.EnvFile == "" {
		spec.EnvFile = ".env"
	}
	if spec.Dockerfile == "" {
		spec.Dockerfile = "Dockerfile"
	}
	if spec.BuildContext == "" {
		spec.BuildContext = "."
	}
	if spec.Strategy == "" {
		spec.Strategy = "docker_save"
	}
	if port, err := strconv.Atoi(strings.TrimSpace(values.Get("ssh_port"))); err == nil && port > 0 {
		spec.SSHPort = port
	} else {
		spec.SSHPort = 22
	}
	return spec
}

const webTplDeploy = `{{define "content"}}
<section class="container">
  <h2 style="margin:0">{{tr "deploy.title"}}</h2>
  <p style="color:#64748b">{{tr "deploy.subtitle"}}</p>
  {{if .Msg}}<article class="alert-ok">{{.Msg}}</article>{{end}}
  {{if .Err}}<article class="alert-err">{{.Err}}</article>{{end}}
  <form method="post" action="/deploy" style="display:grid;gap:.7rem">
    <div class="grid2">
      <label>{{tr "projects.project"}} <input name="proyecto" value="{{.Spec.ProyectoSlug}}" required></label>
      <label>{{tr "deploy.service"}} <input name="servicio" value="{{.Spec.Servicio}}" required></label>
    </div>
    <div class="grid2">
      <label>{{tr "deploy.ssh_host"}} <input name="ssh_host" value="{{.Spec.SSHHost}}" required></label>
      <label>{{tr "deploy.ssh_user"}} <input name="ssh_user" value="{{.Spec.SSHUser}}" required></label>
    </div>
    <div class="grid2">
      <label>{{tr "deploy.ssh_port"}} <input name="ssh_port" type="number" value="{{.Spec.SSHPort}}"></label>
      <label>{{tr "deploy.remote_dir"}} <input name="remote_dir" value="{{.Spec.RemoteDir}}" required></label>
    </div>
    <div class="grid2">
      <label>{{tr "deploy.image"}} <input name="image" value="{{.Spec.Image}}" required></label>
      <label>{{tr "deploy.tag"}} <input name="tag" value="{{.Spec.Tag}}" required></label>
    </div>
    <div class="grid2">
      <label>{{tr "deploy.rollback_tag"}} <input name="rollback_tag" value="{{.Spec.RollbackTag}}"></label>
      <label>{{tr "deploy.strategy"}} <input name="strategy" value="{{.Spec.Strategy}}"></label>
    </div>
    <div class="grid2">
      <label>{{tr "deploy.compose_file"}} <input name="compose_file" value="{{.Spec.ComposeFile}}"></label>
      <label>{{tr "deploy.env_file"}} <input name="env_file" value="{{.Spec.EnvFile}}"></label>
    </div>
    <div class="grid2">
      <label>{{tr "deploy.dockerfile"}} <input name="dockerfile" value="{{.Spec.Dockerfile}}"></label>
      <label>{{tr "deploy.build_context"}} <input name="build_context" value="{{.Spec.BuildContext}}"></label>
    </div>
    <label>{{tr "deploy.healthcheck"}} <input name="healthcheck_cmd" value="{{.Spec.HealthcheckCmd}}"></label>
    <div style="display:flex;gap:1rem;flex-wrap:wrap">
      <label><input type="checkbox" name="build" {{if .Spec.Build}}checked{{end}}> {{tr "deploy.build"}}</label>
      <label><input type="checkbox" name="dry_run" {{if .DryRun}}checked{{end}}> {{tr "deploy.dry_run"}}</label>
      <label><input type="checkbox" name="auto_rollback" {{if .AutoRollback}}checked{{end}}> {{tr "deploy.auto_rollback"}}</label>
    </div>
    <div style="display:flex;gap:.7rem;flex-wrap:wrap">
      <button type="submit" name="accion" value="plan">{{tr "deploy.plan"}}</button>
      <button type="submit" name="accion" value="ejecutar">{{tr "deploy.execute"}}</button>
    </div>
  </form>
</section>

{{if .Plan}}
<section class="container">
  <h3 style="margin:0 0 .5rem 0">{{tr "deploy.plan_title"}}</h3>
  <table>
    <tbody>
      <tr><th>{{tr "deploy.target"}}</th><td>{{.Plan.Target}}</td></tr>
      <tr><th>{{tr "deploy.release_dir"}}</th><td>{{.Plan.ReleaseDir}}</td></tr>
      <tr><th>{{tr "deploy.image_ref"}}</th><td>{{.Plan.ImageRef}}</td></tr>
      <tr><th>{{tr "deploy.strategy"}}</th><td>{{.Plan.Strategy}}</td></tr>
    </tbody>
  </table>
  <h4 style="margin:1rem 0 .5rem 0">{{tr "deploy.steps"}}</h4>
  <ol>
    {{range .Plan.Steps}}
    <li><strong>{{.Name}}</strong>: {{.Description}}<br><code>{{.Shell}}</code></li>
    {{end}}
  </ol>
  {{if .Plan.Warnings}}
  <h4 style="margin:1rem 0 .5rem 0">{{tr "deploy.warnings"}}</h4>
  <ul>
    {{range .Plan.Warnings}}<li>{{.}}</li>{{end}}
  </ul>
  {{end}}
</section>
{{end}}

{{if .Result}}
<section class="container">
  <h3 style="margin:0 0 .5rem 0">{{tr "deploy.result_title"}}</h3>
  <table>
    <tbody>
      <tr><th>{{tr "deploy.failed_step"}}</th><td>{{.Result.FailedStep}}</td></tr>
      <tr><th>{{tr "deploy.rollback_triggered"}}</th><td>{{if .Result.RollbackTriggered}}yes{{else}}no{{end}}</td></tr>
    </tbody>
  </table>
  <h4 style="margin:1rem 0 .5rem 0">{{tr "deploy.executed"}}</h4>
  <ul>
    {{range .Result.Executed}}<li><code>{{.}}</code></li>{{end}}
  </ul>
</section>
{{end}}
{{end}}`
