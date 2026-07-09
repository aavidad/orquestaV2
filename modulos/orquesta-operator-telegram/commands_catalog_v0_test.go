package orquestaoperatortelegram

import "testing"

func TestCommandCatalogV0CubreComandosPublicosYParserV0(t *testing.T) {
	required := map[string]bool{
		CommandStatusV0:  false,
		CommandQueueV0:   false,
		CommandLaunchV0:  false,
		CommandObserveV0: false,
		CommandMessageV0: false,
		CommandStopV0:    false,
		CommandHandoffV0: false,
	}
	aliases := map[string]string{}
	for _, descriptor := range CommandCatalogV0() {
		if _, ok := required[descriptor.Kind]; !ok {
			t.Fatalf("descriptor telegram con kind no inventariado: %+v", descriptor)
		}
		required[descriptor.Kind] = true
		if len(descriptor.Aliases) == 0 {
			t.Fatalf("descriptor telegram sin aliases: %+v", descriptor)
		}
		for _, alias := range descriptor.Aliases {
			if previous := aliases[alias]; previous != "" {
				t.Fatalf("alias telegram duplicado %q en %s y %s", alias, previous, descriptor.Kind)
			}
			aliases[alias] = descriptor.Kind
			command, issues := ParseCommandV0("/" + alias + " " + telegramCommandSampleArgumentsV0(descriptor))
			if len(issues) > 0 || command.Kind != descriptor.Kind {
				t.Fatalf("alias %q no resuelve a %s: command=%+v issues=%+v", alias, descriptor.Kind, command, issues)
			}
		}
	}
	for kind, seen := range required {
		if !seen {
			t.Fatalf("comando publico telegram sin descriptor: %s", kind)
		}
	}
	if !telegramCommandDescriptorByKindV0(CommandStopV0).ConfirmationRequired {
		t.Fatalf("stop debe declarar confirmacion requerida en catalogo")
	}
	if !telegramCommandDescriptorByKindV0(CommandMessageV0).TargetRefRequired ||
		!telegramCommandDescriptorByKindV0(CommandMessageV0).ArgumentsRequired {
		t.Fatalf("director_message debe declarar destino y cuerpo requeridos")
	}
}

func telegramCommandSampleArgumentsV0(descriptor CommandDescriptorV0) string {
	switch descriptor.Kind {
	case CommandLaunchV0:
		return `{"task_ref":"task-ref-1"}`
	case CommandMessageV0:
		return "run-ref-1 mensaje"
	case CommandStopV0:
		return "run-ref-1 confirmar"
	default:
		return "run-ref-1"
	}
}

func telegramCommandDescriptorByKindV0(kind string) CommandDescriptorV0 {
	for _, descriptor := range CommandCatalogV0() {
		if descriptor.Kind == kind {
			return descriptor
		}
	}
	return CommandDescriptorV0{}
}
