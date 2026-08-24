// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package structs

import "time"

// Project is a kanban board owned by a repository, an organization or a user
type Project struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	// enum: ["none", "basic_kanban", "bug_triage"]
	TemplateType string `json:"template_type"`
	// enum: ["text", "images_and_text"]
	CardType        string `json:"card_type"`
	OwnerID         int64  `json:"owner_id"`
	RepoID          int64  `json:"repo_id"`
	CreatorID       int64  `json:"creator_id"`
	IsClosed        bool   `json:"is_closed"`
	NumOpenIssues   int    `json:"num_open_issues"`
	NumClosedIssues int    `json:"num_closed_issues"`
	// swagger:strfmt date-time
	Created time.Time `json:"created_at"`
	// swagger:strfmt date-time
	Updated time.Time `json:"updated_at"`
	// swagger:strfmt date-time
	Closed *time.Time `json:"closed_at"`
}

// ProjectColumn is one column of a project's board
type ProjectColumn struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Color string `json:"color"`
	// the position of the column on the board, ascending
	Sorting int8 `json:"sorting"`
	// issues added to the project without a column land in the default column
	Default   bool  `json:"default"`
	ProjectID int64 `json:"project_id"`
	CreatorID int64 `json:"creator_id"`
	// swagger:strfmt date-time
	Created time.Time `json:"created_at"`
	// swagger:strfmt date-time
	Updated time.Time `json:"updated_at"`
}

// CreateProjectOption options for creating a project
type CreateProjectOption struct {
	// required: true
	Title       string `json:"title" binding:"Required;MaxSize(255)"`
	Description string `json:"description"`
	// the columns the board is created with; "none" creates a board with no columns
	// enum: ["none", "basic_kanban", "bug_triage"]
	TemplateType string `json:"template_type"`
	// enum: ["text", "images_and_text"]
	CardType string `json:"card_type"`
}

// EditProjectOption options for editing a project
type EditProjectOption struct {
	Title       *string `json:"title" binding:"MaxSize(255)"`
	Description *string `json:"description"`
	// enum: ["text", "images_and_text"]
	CardType *string `json:"card_type"`
	// enum: ["open", "closed"]
	State *string `json:"state"`
}

// CreateProjectColumnOption options for creating a project column
type CreateProjectColumnOption struct {
	// required: true
	Title string `json:"title" binding:"Required;MaxSize(255)"`
	// a #rrggbb color, or empty for none
	Color string `json:"color"`
	// make this the column issues land in when added to the project without one
	Default bool `json:"default"`
}

// EditProjectColumnOption options for editing a project column
type EditProjectColumnOption struct {
	Title *string `json:"title" binding:"MaxSize(255)"`
	Color *string `json:"color"`
	// setting this to true makes the column the default one; false is ignored,
	// because a project always has exactly one default column
	Default *bool `json:"default"`
}

// MovedProjectColumn is one column's new position on the board
type MovedProjectColumn struct {
	// required: true
	ID int64 `json:"id"`
	// required: true
	Sorting int64 `json:"sorting"`
}

// MoveProjectColumnsOption options for reordering the columns of a project
type MoveProjectColumnsOption struct {
	// required: true
	Columns []MovedProjectColumn `json:"columns"`
}

// AddProjectIssueOption options for adding an issue to a project column
type AddProjectIssueOption struct {
	// the index of the issue in its repository, as used by /issues/{index}
	// required: true
	Index int64 `json:"index" binding:"Required"`
}

// MovedProjectIssue is one issue's new position within a column
type MovedProjectIssue struct {
	// required: true
	Index int64 `json:"index"`
	// required: true
	Sorting int64 `json:"sorting"`
}

// MoveProjectIssuesOption options for sorting the issues of a project column
type MoveProjectIssuesOption struct {
	// required: true
	Issues []MovedProjectIssue `json:"issues"`
}
