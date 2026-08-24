// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"fmt"
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	issues_model "forgejo.org/models/issues"
	project_model "forgejo.org/models/project"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIRepoProject(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1, RepoID: repo.ID})

	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)
	base := fmt.Sprintf("/api/v1/repos/%s/%s/projects", owner.Name, repo.Name)

	t.Run("a token is required to write", func(t *testing.T) {
		req := NewRequestWithJSON(t, "POST", base, structs.CreateProjectOption{Title: "no token"})
		MakeRequest(t, req, http.StatusUnauthorized)
	})

	t.Run("list the projects the fixtures give repo 1", func(t *testing.T) {
		req := NewRequest(t, "GET", base).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var projects []*structs.Project
		DecodeJSON(t, resp, &projects)
		require.Len(t, projects, 1)
		assert.Equal(t, "First project", projects[0].Title)
		assert.False(t, projects[0].IsClosed)
	})

	var project structs.Project

	t.Run("create a project", func(t *testing.T) {
		req := NewRequestWithJSON(t, "POST", base, structs.CreateProjectOption{
			Title:        "API kanban",
			Description:  "built over REST",
			TemplateType: "none",
			CardType:     "text",
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusCreated)

		DecodeJSON(t, resp, &project)
		assert.Equal(t, "API kanban", project.Title)
		assert.Equal(t, "none", project.TemplateType)
		assert.Equal(t, "text", project.CardType)
		assert.Equal(t, repo.ID, project.RepoID)

		// the "none" template means the board starts with no columns at all
		req = NewRequest(t, "GET", fmt.Sprintf("%s/%d/columns", base, project.ID)).AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusOK)

		var columns []*structs.ProjectColumn
		DecodeJSON(t, resp, &columns)
		assert.Empty(t, columns)
	})

	t.Run("reject an unknown card type", func(t *testing.T) {
		req := NewRequestWithJSON(t, "POST", base, structs.CreateProjectOption{
			Title:    "bad card type",
			CardType: "hologram",
		}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusUnprocessableEntity)
	})

	var todo, doing structs.ProjectColumn
	columnsURL := fmt.Sprintf("%s/%d/columns", base, project.ID)

	t.Run("add columns to the board", func(t *testing.T) {
		req := NewRequestWithJSON(t, "POST", columnsURL, structs.CreateProjectColumnOption{
			Title:   "To do",
			Color:   "#112233",
			Default: true,
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusCreated)
		DecodeJSON(t, resp, &todo)
		assert.Equal(t, "To do", todo.Title)
		assert.Equal(t, "#112233", todo.Color)
		assert.True(t, todo.Default)

		req = NewRequestWithJSON(t, "POST", columnsURL, structs.CreateProjectColumnOption{
			Title: "Doing",
		}).AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusCreated)
		DecodeJSON(t, resp, &doing)
		assert.Equal(t, "Doing", doing.Title)
		assert.False(t, doing.Default)

		// listed in board order, which is the order they were added in
		req = NewRequest(t, "GET", columnsURL).AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusOK)

		var columns []*structs.ProjectColumn
		DecodeJSON(t, resp, &columns)
		require.Len(t, columns, 2)
		assert.Equal(t, "To do", columns[0].Title)
		assert.Equal(t, "Doing", columns[1].Title)
	})

	t.Run("refuse a colour that is not #rrggbb", func(t *testing.T) {
		req := NewRequestWithJSON(t, "POST", columnsURL, structs.CreateProjectColumnOption{
			Title: "chartreuse",
			Color: "chartreuse",
		}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusUnprocessableEntity)
	})

	t.Run("rename a column", func(t *testing.T) {
		title := "In progress"
		req := NewRequestWithJSON(t, "PATCH", fmt.Sprintf("%s/%d", columnsURL, doing.ID), structs.EditProjectColumnOption{
			Title: &title,
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var edited structs.ProjectColumn
		DecodeJSON(t, resp, &edited)
		assert.Equal(t, "In progress", edited.Title)
	})

	t.Run("put an issue on a column and take it off again", func(t *testing.T) {
		req := NewRequestWithJSON(t, "POST", fmt.Sprintf("%s/%d/issues", columnsURL, todo.ID), structs.AddProjectIssueOption{
			Index: issue.Index,
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusCreated)

		var carded structs.Issue
		DecodeJSON(t, resp, &carded)
		assert.Equal(t, issue.Index, carded.Index)

		req = NewRequest(t, "GET", fmt.Sprintf("%s/%d/issues", columnsURL, todo.ID)).AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusOK)

		var issues []*structs.Issue
		DecodeJSON(t, resp, &issues)
		require.Len(t, issues, 1)
		assert.Equal(t, issue.Index, issues[0].Index)

		// moving it to the other column takes it off this one
		req = NewRequestWithJSON(t, "POST", fmt.Sprintf("%s/%d/issues", columnsURL, doing.ID), structs.AddProjectIssueOption{
			Index: issue.Index,
		}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusCreated)

		req = NewRequest(t, "GET", fmt.Sprintf("%s/%d/issues", columnsURL, todo.ID)).AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusOK)
		DecodeJSON(t, resp, &issues)
		assert.Empty(t, issues)

		req = NewRequest(t, "DELETE", fmt.Sprintf("%s/%d/issues/%d", base, project.ID, issue.Index)).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNoContent)

		req = NewRequest(t, "GET", fmt.Sprintf("%s/%d/issues", columnsURL, doing.ID)).AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusOK)
		DecodeJSON(t, resp, &issues)
		assert.Empty(t, issues)
	})

	t.Run("reorder the whole board", func(t *testing.T) {
		req := NewRequestWithJSON(t, "PATCH", columnsURL, structs.MoveProjectColumnsOption{
			Columns: []structs.MovedProjectColumn{
				{ID: doing.ID, Sorting: 0},
				{ID: todo.ID, Sorting: 1},
			},
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var columns []*structs.ProjectColumn
		DecodeJSON(t, resp, &columns)
		require.Len(t, columns, 2)
		assert.Equal(t, doing.ID, columns[0].ID)
		assert.Equal(t, todo.ID, columns[1].ID)
	})

	t.Run("refuse a reorder that leaves a column out", func(t *testing.T) {
		req := NewRequestWithJSON(t, "PATCH", columnsURL, structs.MoveProjectColumnsOption{
			Columns: []structs.MovedProjectColumn{{ID: todo.ID, Sorting: 0}},
		}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusUnprocessableEntity)
	})

	t.Run("refuse a column that belongs to another project", func(t *testing.T) {
		other := unittest.AssertExistsAndLoadBean(t, &project_model.Column{ID: 1})
		require.NotEqual(t, project.ID, other.ProjectID)

		req := NewRequest(t, "GET", fmt.Sprintf("%s/%d", columnsURL, other.ID)).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNotFound)
	})

	t.Run("close the project, then delete it", func(t *testing.T) {
		state := "closed"
		req := NewRequestWithJSON(t, "PATCH", fmt.Sprintf("%s/%d", base, project.ID), structs.EditProjectOption{
			State: &state,
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var closed structs.Project
		DecodeJSON(t, resp, &closed)
		assert.True(t, closed.IsClosed)

		req = NewRequest(t, "DELETE", fmt.Sprintf("%s/%d", base, project.ID)).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNoContent)

		req = NewRequest(t, "GET", fmt.Sprintf("%s/%d", base, project.ID)).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNotFound)
	})

	t.Run("a repository with the projects unit disabled has no endpoints", func(t *testing.T) {
		disabled := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 4})
		disabledOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: disabled.OwnerID})

		req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects", disabledOwner.Name, disabled.Name)).
			AddTokenAuth(getTokenForLoggedInUser(t, loginUser(t, disabledOwner.Name), auth_model.AccessTokenScopeWriteRepository))
		MakeRequest(t, req, http.StatusNotFound)
	})
}
