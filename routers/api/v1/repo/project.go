// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package repo

import (
	"errors"
	"net/http"
	"strings"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	project_model "forgejo.org/models/project"
	"forgejo.org/modules/optional"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/util"
	"forgejo.org/modules/web"
	"forgejo.org/routers/api/v1/utils"
	"forgejo.org/services/context"
	"forgejo.org/services/convert"
)

// ListProjects list a repository's projects
func ListProjects(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/projects project repoListProjects
	// ---
	// summary: List a repository's projects
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: state
	//   in: query
	//   description: 'project state, recognized values are open, closed and all. Defaults to "open"'
	//   type: string
	// - name: q
	//   in: query
	//   description: filter by title
	//   type: string
	// - name: sort
	//   in: query
	//   description: 'sort order, recognized values are oldest, recentupdate and leastupdate. Defaults to newest'
	//   type: string
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/ProjectList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	var isClosed optional.Option[bool]
	switch strings.ToLower(ctx.FormTrim("state")) {
	case "closed":
		isClosed = optional.Some(true)
	case "all":
		isClosed = optional.None[bool]()
	default:
		isClosed = optional.Some(false)
	}

	projects, total, err := db.FindAndCount[project_model.Project](ctx, project_model.SearchOptions{
		ListOptions: utils.GetListOptions(ctx),
		RepoID:      ctx.Repo().Repository.ID,
		IsClosed:    isClosed,
		OrderBy:     project_model.GetSearchOrderByBySortType(ctx.FormTrim("sort")),
		Type:        project_model.TypeRepository,
		Title:       ctx.FormTrim("q"),
	})
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "db.FindAndCount[project_model.Project]", err)
		return
	}

	apiProjects := make([]*api.Project, len(projects))
	for i := range projects {
		apiProjects[i] = convert.ToAPIProject(ctx, projects[i])
	}

	ctx.SetTotalCountHeader(total)
	ctx.JSON(http.StatusOK, &apiProjects)
}

// GetProject get one of a repository's projects
func GetProject(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/projects/{id} project repoGetProject
	// ---
	// summary: Get a project
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/Project"
	//   "404":
	//     "$ref": "#/responses/notFound"

	project := getRepoProject(ctx)
	if ctx.Written() {
		return
	}

	ctx.JSON(http.StatusOK, convert.ToAPIProject(ctx, project))
}

// CreateProject create a project in a repository
func CreateProject(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/projects project repoCreateProject
	// ---
	// summary: Create a project
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/CreateProjectOption"
	// responses:
	//   "201":
	//     "$ref": "#/responses/Project"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"

	form := web.GetForm(ctx).(*api.CreateProjectOption)

	templateType := project_model.TemplateTypeNone
	if form.TemplateType != "" {
		parsed, err := convert.ToProjectTemplateType(form.TemplateType)
		if err != nil {
			ctx.Error(http.StatusUnprocessableEntity, "ToProjectTemplateType", err)
			return
		}
		templateType = parsed
	}

	cardType := project_model.CardTypeTextOnly
	if form.CardType != "" {
		parsed, err := convert.ToProjectCardType(form.CardType)
		if err != nil {
			ctx.Error(http.StatusUnprocessableEntity, "ToProjectCardType", err)
			return
		}
		cardType = parsed
	}

	project := &project_model.Project{
		RepoID:       ctx.Repo().Repository.ID,
		OwnerID:      ctx.Repo().Repository.OwnerID,
		Title:        form.Title,
		Description:  form.Description,
		CreatorID:    ctx.Doer().ID,
		TemplateType: templateType,
		CardType:     cardType,
		Type:         project_model.TypeRepository,
	}

	if err := project_model.NewProject(ctx, project); err != nil {
		ctx.Error(http.StatusInternalServerError, "NewProject", err)
		return
	}

	ctx.JSON(http.StatusCreated, convert.ToAPIProject(ctx, project))
}

// EditProject edit one of a repository's projects
func EditProject(ctx *context.APIContext) {
	// swagger:operation PATCH /repos/{owner}/{repo}/projects/{id} project repoEditProject
	// ---
	// summary: Update a project
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/EditProjectOption"
	// responses:
	//   "200":
	//     "$ref": "#/responses/Project"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"

	form := web.GetForm(ctx).(*api.EditProjectOption)
	project := getRepoProject(ctx)
	if ctx.Written() {
		return
	}

	if form.Title != nil {
		project.Title = *form.Title
	}
	if form.Description != nil {
		project.Description = *form.Description
	}
	if form.CardType != nil {
		cardType, err := convert.ToProjectCardType(*form.CardType)
		if err != nil {
			ctx.Error(http.StatusUnprocessableEntity, "ToProjectCardType", err)
			return
		}
		project.CardType = cardType
	}

	if err := project_model.UpdateProject(ctx, project); err != nil {
		ctx.Error(http.StatusInternalServerError, "UpdateProject", err)
		return
	}

	// The status lives in its own column and its own counters, so it is changed
	// through ChangeProjectStatus rather than written with the fields above.
	if form.State != nil {
		isClosed := *form.State == string(api.StateClosed)
		if isClosed != project.IsClosed {
			if err := project_model.ChangeProjectStatus(ctx, project, isClosed); err != nil {
				ctx.Error(http.StatusInternalServerError, "ChangeProjectStatus", err)
				return
			}
		}
	}

	ctx.JSON(http.StatusOK, convert.ToAPIProject(ctx, project))
}

// DeleteProject delete one of a repository's projects
func DeleteProject(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/projects/{id} project repoDeleteProject
	// ---
	// summary: Delete a project
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "404":
	//     "$ref": "#/responses/notFound"

	project := getRepoProject(ctx)
	if ctx.Written() {
		return
	}

	if err := project_model.DeleteProjectByID(ctx, project.ID); err != nil {
		ctx.Error(http.StatusInternalServerError, "DeleteProjectByID", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// ListProjectColumns list the columns of a project
func ListProjectColumns(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/projects/{id}/columns project repoListProjectColumns
	// ---
	// summary: List a project's columns, in board order
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ProjectColumnList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	project := getRepoProject(ctx)
	if ctx.Written() {
		return
	}

	columns, err := project.GetColumns(ctx)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetColumns", err)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToAPIProjectColumnList(columns))
}

// GetProjectColumn get one column of a project
func GetProjectColumn(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/projects/{id}/columns/{column_id} project repoGetProjectColumn
	// ---
	// summary: Get a project column
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// - name: column_id
	//   in: path
	//   description: id of the column
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ProjectColumn"
	//   "404":
	//     "$ref": "#/responses/notFound"

	_, column := getRepoProjectColumn(ctx)
	if ctx.Written() {
		return
	}

	ctx.JSON(http.StatusOK, convert.ToAPIProjectColumn(column))
}

// CreateProjectColumn add a column to a project
func CreateProjectColumn(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/projects/{id}/columns project repoCreateProjectColumn
	// ---
	// summary: Add a column to a project
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/CreateProjectColumnOption"
	// responses:
	//   "201":
	//     "$ref": "#/responses/ProjectColumn"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"

	form := web.GetForm(ctx).(*api.CreateProjectColumnOption)
	project := getRepoProject(ctx)
	if ctx.Written() {
		return
	}

	column := &project_model.Column{
		ProjectID: project.ID,
		Title:     form.Title,
		Color:     form.Color,
		CreatorID: ctx.Doer().ID,
	}

	if form.Color != "" && !project_model.ColumnColorPattern.MatchString(form.Color) {
		ctx.Error(http.StatusUnprocessableEntity, "ColumnColorPattern", errors.New("color must be a #rrggbb value"))
		return
	}

	// Every remaining way NewColumn fails is the caller's: the colour above, or
	// the project already holding the maximum number of columns.
	if err := project_model.NewColumn(ctx, column); err != nil {
		ctx.Error(http.StatusUnprocessableEntity, "NewColumn", err)
		return
	}

	if form.Default {
		if err := project_model.SetDefaultColumn(ctx, project.ID, column.ID); err != nil {
			ctx.Error(http.StatusInternalServerError, "SetDefaultColumn", err)
			return
		}
		column.Default = true
	}

	ctx.JSON(http.StatusCreated, convert.ToAPIProjectColumn(column))
}

// EditProjectColumn edit one column of a project
func EditProjectColumn(ctx *context.APIContext) {
	// swagger:operation PATCH /repos/{owner}/{repo}/projects/{id}/columns/{column_id} project repoEditProjectColumn
	// ---
	// summary: Update a project column
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// - name: column_id
	//   in: path
	//   description: id of the column
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/EditProjectColumnOption"
	// responses:
	//   "200":
	//     "$ref": "#/responses/ProjectColumn"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"

	form := web.GetForm(ctx).(*api.EditProjectColumnOption)
	project, column := getRepoProjectColumn(ctx)
	if ctx.Written() {
		return
	}

	if form.Title != nil {
		column.Title = *form.Title
	}
	if form.Color != nil {
		column.Color = *form.Color
	}

	if err := project_model.UpdateColumn(ctx, column); err != nil {
		ctx.Error(http.StatusUnprocessableEntity, "UpdateColumn", err)
		return
	}

	// A project has exactly one default column, so this only ever sets one --
	// clearing it would leave the project with none.
	if form.Default != nil && *form.Default {
		if err := project_model.SetDefaultColumn(ctx, project.ID, column.ID); err != nil {
			ctx.Error(http.StatusInternalServerError, "SetDefaultColumn", err)
			return
		}
		column.Default = true
	}

	ctx.JSON(http.StatusOK, convert.ToAPIProjectColumn(column))
}

// DeleteProjectColumn remove one column from a project
func DeleteProjectColumn(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/projects/{id}/columns/{column_id} project repoDeleteProjectColumn
	// ---
	// summary: Delete a project column; its issues move to the project's default column
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// - name: column_id
	//   in: path
	//   description: id of the column
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "404":
	//     "$ref": "#/responses/notFound"

	_, column := getRepoProjectColumn(ctx)
	if ctx.Written() {
		return
	}

	if err := project_model.DeleteColumnByID(ctx, column.ID); err != nil {
		ctx.Error(http.StatusInternalServerError, "DeleteColumnByID", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// MoveProjectColumns reorder every column of a project
func MoveProjectColumns(ctx *context.APIContext) {
	// swagger:operation PATCH /repos/{owner}/{repo}/projects/{id}/columns project repoMoveProjectColumns
	// ---
	// summary: Reorder a project's columns; every column of the project must be listed
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/MoveProjectColumnsOption"
	// responses:
	//   "200":
	//     "$ref": "#/responses/ProjectColumnList"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"

	form := web.GetForm(ctx).(*api.MoveProjectColumnsOption)
	project := getRepoProject(ctx)
	if ctx.Written() {
		return
	}

	sortedColumnIDs := make(map[int64]int64, len(form.Columns))
	for _, column := range form.Columns {
		sortedColumnIDs[column.Sorting] = column.ID
	}

	if err := project_model.MoveColumnsOnProject(ctx, project, sortedColumnIDs); err != nil {
		ctx.Error(http.StatusUnprocessableEntity, "MoveColumnsOnProject", err)
		return
	}

	columns, err := project.GetColumns(ctx)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetColumns", err)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToAPIProjectColumnList(columns))
}

// ListProjectColumnIssues list the issues of one project column
func ListProjectColumnIssues(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/projects/{id}/columns/{column_id}/issues project repoListProjectColumnIssues
	// ---
	// summary: List the issues on a project column, in card order
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// - name: column_id
	//   in: path
	//   description: id of the column
	//   type: integer
	//   format: int64
	//   required: true
	// - name: state
	//   in: query
	//   description: 'issue state, recognized values are open, closed and all. Defaults to "all"'
	//   type: string
	// responses:
	//   "200":
	//     "$ref": "#/responses/IssueList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	_, column := getRepoProjectColumn(ctx)
	if ctx.Written() {
		return
	}

	var isClosed optional.Option[bool]
	switch strings.ToLower(ctx.FormTrim("state")) {
	case "open":
		isClosed = optional.Some(false)
	case "closed":
		isClosed = optional.Some(true)
	}

	issues, err := issues_model.LoadIssuesFromColumn(ctx, column, ctx.Doer(), nil, isClosed)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "LoadIssuesFromColumn", err)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToAPIIssueList(ctx, ctx.Doer(), issues))
}

// AddIssueToProjectColumn put an issue on a project column
func AddIssueToProjectColumn(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/projects/{id}/columns/{column_id}/issues project repoAddIssueToProjectColumn
	// ---
	// summary: Add an issue to a project column, moving it there if it is already on the project
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// - name: column_id
	//   in: path
	//   description: id of the column
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/AddProjectIssueOption"
	// responses:
	//   "201":
	//     "$ref": "#/responses/Issue"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	form := web.GetForm(ctx).(*api.AddProjectIssueOption)
	project, column := getRepoProjectColumn(ctx)
	if ctx.Written() {
		return
	}

	issue := getRepoIssueByIndex(ctx, form.Index)
	if ctx.Written() {
		return
	}

	if err := issues_model.IssueAssignOrRemoveProject(ctx, issue, ctx.Doer(), project.ID, column.ID); err != nil {
		if errors.Is(err, util.ErrPermissionDenied) {
			ctx.Error(http.StatusForbidden, "IssueAssignOrRemoveProject", err)
			return
		}
		ctx.Error(http.StatusInternalServerError, "IssueAssignOrRemoveProject", err)
		return
	}

	ctx.JSON(http.StatusCreated, convert.ToAPIIssue(ctx, ctx.Doer(), issue))
}

// MoveProjectColumnIssues sort the issues of one project column
func MoveProjectColumnIssues(ctx *context.APIContext) {
	// swagger:operation PATCH /repos/{owner}/{repo}/projects/{id}/columns/{column_id}/issues project repoMoveProjectColumnIssues
	// ---
	// summary: Sort the issues already on a project column
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// - name: column_id
	//   in: path
	//   description: id of the column
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/MoveProjectIssuesOption"
	// responses:
	//   "200":
	//     "$ref": "#/responses/IssueList"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"

	form := web.GetForm(ctx).(*api.MoveProjectIssuesOption)
	_, column := getRepoProjectColumn(ctx)
	if ctx.Written() {
		return
	}

	sortedIssueIDs := make(map[int64]int64, len(form.Issues))
	for _, moved := range form.Issues {
		issue := getRepoIssueByIndex(ctx, moved.Index)
		if ctx.Written() {
			return
		}
		sortedIssueIDs[moved.Sorting] = issue.ID
	}

	if err := project_model.MoveIssuesOnProjectColumn(ctx, column, sortedIssueIDs); err != nil {
		ctx.Error(http.StatusUnprocessableEntity, "MoveIssuesOnProjectColumn", err)
		return
	}

	issues, err := issues_model.LoadIssuesFromColumn(ctx, column, ctx.Doer(), nil, optional.None[bool]())
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "LoadIssuesFromColumn", err)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToAPIIssueList(ctx, ctx.Doer(), issues))
}

// RemoveIssueFromProject take an issue off a project
func RemoveIssueFromProject(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/projects/{id}/issues/{index} project repoRemoveIssueFromProject
	// ---
	// summary: Remove an issue from a project, leaving the issue itself alone
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// - name: index
	//   in: path
	//   description: index of the issue in the repository
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	project := getRepoProject(ctx)
	if ctx.Written() {
		return
	}

	issue := getRepoIssueByIndex(ctx, ctx.ParamsInt64(":index"))
	if ctx.Written() {
		return
	}

	if err := issue.LoadProject(ctx); err != nil {
		ctx.Error(http.StatusInternalServerError, "LoadProject", err)
		return
	}
	if issue.Project == nil || issue.Project.ID != project.ID {
		ctx.NotFound()
		return
	}

	if err := issues_model.IssueAssignOrRemoveProject(ctx, issue, ctx.Doer(), 0, 0); err != nil {
		if errors.Is(err, util.ErrPermissionDenied) {
			ctx.Error(http.StatusForbidden, "IssueAssignOrRemoveProject", err)
			return
		}
		ctx.Error(http.StatusInternalServerError, "IssueAssignOrRemoveProject", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// getRepoProject reads :id and refuses a project belonging to another repository
func getRepoProject(ctx *context.APIContext) *project_model.Project {
	project, err := project_model.GetProjectForRepoByID(ctx, ctx.Repo().Repository.ID, ctx.ParamsInt64(":id"))
	if err != nil {
		ctx.NotFoundOrServerError("GetProjectForRepoByID", project_model.IsErrProjectNotExist, err)
		return nil
	}
	return project
}

// getRepoProjectColumn reads :id and :column_id, refusing a column that belongs
// to another project even when both exist
func getRepoProjectColumn(ctx *context.APIContext) (*project_model.Project, *project_model.Column) {
	project := getRepoProject(ctx)
	if ctx.Written() {
		return nil, nil
	}

	column, err := project_model.GetColumn(ctx, ctx.ParamsInt64(":column_id"))
	if err != nil {
		ctx.NotFoundOrServerError("GetColumn", project_model.IsErrProjectColumnNotExist, err)
		return nil, nil
	}

	if column.ProjectID != project.ID {
		ctx.NotFound()
		return nil, nil
	}

	return project, column
}

func getRepoIssueByIndex(ctx *context.APIContext, index int64) *issues_model.Issue {
	issue, err := issues_model.GetIssueByIndex(ctx, ctx.Repo().Repository.ID, index)
	if err != nil {
		ctx.NotFoundOrServerError("GetIssueByIndex", issues_model.IsErrIssueNotExist, err)
		return nil
	}
	return issue
}
