package controller

import "github.com/Saad7890-web/orbit/internal/models"

type DesiredState struct {
	Stack      models.Stack
	ConfigHash string
}

func NewDesiredState(stack models.Stack, configHash string) DesiredState {
	return DesiredState{
		Stack:      stack,
		ConfigHash: configHash,
	}
}

func (d DesiredState) ServiceByName(name string) (*models.Service, bool) {
	for i := range d.Stack.Services {
		if d.Stack.Services[i].Name == name {
			return &d.Stack.Services[i], true
		}
	}
	return nil, false
}

func (d DesiredState) JobByName(name string) (*models.Job, bool) {
	for i := range d.Stack.Jobs {
		if d.Stack.Jobs[i].Name == name {
			return &d.Stack.Jobs[i], true
		}
	}
	return nil, false
}

func (d DesiredState) TriggerByName(name string) (*models.Trigger, bool) {
	for i := range d.Stack.Triggers {
		if d.Stack.Triggers[i].Name == name {
			return &d.Stack.Triggers[i], true
		}
	}
	return nil, false
}