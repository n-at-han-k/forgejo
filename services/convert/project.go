// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package convert

import (
	"context"

	project_model "forgejo.org/models/project"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/util"
)

var (
	projectTemplateTypeNames = map[project_model.TemplateType]string{
		project_model.TemplateTypeNone:        "none",
		project_model.TemplateTypeBasicKanban: "basic_kanban",
		project_model.TemplateTypeBugTriage:   "bug_triage",
	}

	projectCardTypeNames = map[project_model.CardType]string{
		project_model.CardTypeTextOnly:      "text",
		project_model.CardTypeImagesAndText: "images_and_text",
	}
)

// ToAPIProjectTemplateType returns the API name of a project template type
func ToAPIProjectTemplateType(templateType project_model.TemplateType) string {
	return projectTemplateTypeNames[templateType]
}

// ToAPIProjectCardType returns the API name of a project card type
func ToAPIProjectCardType(cardType project_model.CardType) string {
	return projectCardTypeNames[cardType]
}

// ToProjectTemplateType converts an API template type name to its model value
func ToProjectTemplateType(name string) (project_model.TemplateType, error) {
	for templateType, candidate := range projectTemplateTypeNames {
		if candidate == name {
			return templateType, nil
		}
	}
	return project_model.TemplateTypeNone, util.NewInvalidArgumentErrorf("unknown project template type %q", name)
}

// ToProjectCardType converts an API card type name to its model value
func ToProjectCardType(name string) (project_model.CardType, error) {
	for cardType, candidate := range projectCardTypeNames {
		if candidate == name {
			return cardType, nil
		}
	}
	return project_model.CardTypeTextOnly, util.NewInvalidArgumentErrorf("unknown project card type %q", name)
}

// ToAPIProject converts a project to its API representation
func ToAPIProject(ctx context.Context, project *project_model.Project) *api.Project {
	apiProject := &api.Project{
		ID:              project.ID,
		Title:           project.Title,
		Description:     project.Description,
		TemplateType:    ToAPIProjectTemplateType(project.TemplateType),
		CardType:        ToAPIProjectCardType(project.CardType),
		OwnerID:         project.OwnerID,
		RepoID:          project.RepoID,
		CreatorID:       project.CreatorID,
		IsClosed:        project.IsClosed,
		NumOpenIssues:   project.NumOpenIssues(ctx),
		NumClosedIssues: project.NumClosedIssues(ctx),
		Created:         project.CreatedUnix.AsTime(),
		Updated:         project.UpdatedUnix.AsTime(),
	}

	if project.IsClosed {
		closed := project.ClosedDateUnix.AsTime()
		apiProject.Closed = &closed
	}

	return apiProject
}

// ToAPIProjectColumn converts a project column to its API representation
func ToAPIProjectColumn(column *project_model.Column) *api.ProjectColumn {
	return &api.ProjectColumn{
		ID:        column.ID,
		Title:     column.Title,
		Color:     column.Color,
		Sorting:   column.Sorting,
		Default:   column.Default,
		ProjectID: column.ProjectID,
		CreatorID: column.CreatorID,
		Created:   column.CreatedUnix.AsTime(),
		Updated:   column.UpdatedUnix.AsTime(),
	}
}

// ToAPIProjectColumnList converts a list of project columns
func ToAPIProjectColumnList(columns project_model.ColumnList) []*api.ProjectColumn {
	apiColumns := make([]*api.ProjectColumn, len(columns))
	for i := range columns {
		apiColumns[i] = ToAPIProjectColumn(columns[i])
	}
	return apiColumns
}
