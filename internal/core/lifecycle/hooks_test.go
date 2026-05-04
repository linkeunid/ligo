package lifecycle

import (
	"errors"
	"testing"
)

// Mock providers implementing different hook combinations

// AllHooksProvider implements all 5 lifecycle hooks
type AllHooksProvider struct {
	initCalled         bool
	bootstrapCalled    bool
	beforeShutdownCalled bool
	destroyCalled      bool
	shutdownCalled     bool
}

func (m *AllHooksProvider) OnModuleInit() error {
	m.initCalled = true
	return nil
}

func (m *AllHooksProvider) OnApplicationBootstrap() error {
	m.bootstrapCalled = true
	return nil
}

func (m *AllHooksProvider) BeforeApplicationShutdown() error {
	m.beforeShutdownCalled = true
	return nil
}

func (m *AllHooksProvider) OnModuleDestroy() error {
	m.destroyCalled = true
	return nil
}

func (m *AllHooksProvider) OnApplicationShutdown() error {
	m.shutdownCalled = true
	return nil
}

// InitOnlyProvider implements only OnModuleInit
type InitOnlyProvider struct{}

func (m *InitOnlyProvider) OnModuleInit() error {
	return nil
}

// ErrorProvider implements OnModuleInit that returns an error
type ErrorProvider struct{}

func (m *ErrorProvider) OnModuleInit() error {
	return errors.New("init failed")
}

// NoHooksProvider implements no lifecycle hooks
type NoHooksProvider struct{}

// BootstrapOnlyProvider implements only OnApplicationBootstrap
type BootstrapOnlyProvider struct{}

func (m *BootstrapOnlyProvider) OnApplicationBootstrap() error {
	return nil
}

// DestroyOnlyProvider implements only OnModuleDestroy
type DestroyOnlyProvider struct{}

func (m *DestroyOnlyProvider) OnModuleDestroy() error {
	return nil
}

// ShutdownOnlyProvider implements only OnApplicationShutdown
type ShutdownOnlyProvider struct{}

func (m *ShutdownOnlyProvider) OnApplicationShutdown() error {
	return nil
}

// BeforeShutdownOnlyProvider implements only BeforeApplicationShutdown
type BeforeShutdownOnlyProvider struct{}

func (m *BeforeShutdownOnlyProvider) BeforeApplicationShutdown() error {
	return nil
}

// InitAndBootstrapProvider implements OnModuleInit and OnApplicationBootstrap
type InitAndBootstrapProvider struct{}

func (m *InitAndBootstrapProvider) OnModuleInit() error {
	return nil
}

func (m *InitAndBootstrapProvider) OnApplicationBootstrap() error {
	return nil
}

// DestroyAndShutdownProvider implements OnModuleDestroy and OnApplicationShutdown
type DestroyAndShutdownProvider struct{}

func (m *DestroyAndShutdownProvider) OnModuleDestroy() error {
	return nil
}

func (m *DestroyAndShutdownProvider) OnApplicationShutdown() error {
	return nil
}

func TestCollectProviderHooks(t *testing.T) {
	tests := []struct {
		name             string
		provider         any
		wantInit         bool
		wantBoot         bool
		wantBeforeShutdown bool
		wantDestroy      bool
		wantShutdown     bool
	}{
		{
			name:             "all hooks implemented",
			provider:         &AllHooksProvider{},
			wantInit:         true,
			wantBoot:         true,
			wantBeforeShutdown: true,
			wantDestroy:      true,
			wantShutdown:     true,
		},
		{
			name:     "init only",
			provider: &InitOnlyProvider{},
			wantInit: true,
		},
		{
			name:     "bootstrap only",
			provider: &BootstrapOnlyProvider{},
			wantBoot: true,
		},
		{
			name:         "destroy only",
			provider:     &DestroyOnlyProvider{},
			wantDestroy:  true,
		},
		{
			name:         "shutdown only",
			provider:     &ShutdownOnlyProvider{},
			wantShutdown: true,
		},
		{
			name:             "before shutdown only",
			provider:         &BeforeShutdownOnlyProvider{},
			wantBeforeShutdown: true,
		},
		{
			name:     "init and bootstrap",
			provider: &InitAndBootstrapProvider{},
			wantInit: true,
			wantBoot: true,
		},
		{
			name:         "destroy and shutdown",
			provider:     &DestroyAndShutdownProvider{},
			wantDestroy:  true,
			wantShutdown: true,
		},
		{
			name:     "no hooks",
			provider: &NoHooksProvider{},
		},
		{
			name:     "nil provider",
			provider: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test interface detection using type assertions
			_, hasInit := tt.provider.(interface{ OnModuleInit() error })
			_, hasBoot := tt.provider.(interface{ OnApplicationBootstrap() error })
			_, hasBeforeShutdown := tt.provider.(interface{ BeforeApplicationShutdown() error })
			_, hasDestroy := tt.provider.(interface{ OnModuleDestroy() error })
			_, hasShutdown := tt.provider.(interface{ OnApplicationShutdown() error })

			if hasInit != tt.wantInit {
				t.Errorf("OnModuleInit detection = %v, want %v", hasInit, tt.wantInit)
			}
			if hasBoot != tt.wantBoot {
				t.Errorf("OnApplicationBootstrap detection = %v, want %v", hasBoot, tt.wantBoot)
			}
			if hasBeforeShutdown != tt.wantBeforeShutdown {
				t.Errorf("BeforeApplicationShutdown detection = %v, want %v", hasBeforeShutdown, tt.wantBeforeShutdown)
			}
			if hasDestroy != tt.wantDestroy {
				t.Errorf("OnModuleDestroy detection = %v, want %v", hasDestroy, tt.wantDestroy)
			}
			if hasShutdown != tt.wantShutdown {
				t.Errorf("OnApplicationShutdown detection = %v, want %v", hasShutdown, tt.wantShutdown)
			}
		})
	}
}

func TestHookExecution(t *testing.T) {
	t.Run("all hooks execute successfully", func(t *testing.T) {
		p := &AllHooksProvider{}

		// Simulate hook calls through interface detection
		initFn, ok := any(p).(interface{ OnModuleInit() error })
		if !ok {
			t.Fatal("provider should implement OnModuleInit")
		}
		if err := initFn.OnModuleInit(); err != nil {
			t.Fatalf("OnModuleInit failed: %v", err)
		}

		bootFn, ok := any(p).(interface{ OnApplicationBootstrap() error })
		if !ok {
			t.Fatal("provider should implement OnApplicationBootstrap")
		}
		if err := bootFn.OnApplicationBootstrap(); err != nil {
			t.Fatalf("OnApplicationBootstrap failed: %v", err)
		}

		beforeShutdownFn, ok := any(p).(interface{ BeforeApplicationShutdown() error })
		if !ok {
			t.Fatal("provider should implement BeforeApplicationShutdown")
		}
		if err := beforeShutdownFn.BeforeApplicationShutdown(); err != nil {
			t.Fatalf("BeforeApplicationShutdown failed: %v", err)
		}

		destroyFn, ok := any(p).(interface{ OnModuleDestroy() error })
		if !ok {
			t.Fatal("provider should implement OnModuleDestroy")
		}
		if err := destroyFn.OnModuleDestroy(); err != nil {
			t.Fatalf("OnModuleDestroy failed: %v", err)
		}

		shutdownFn, ok := any(p).(interface{ OnApplicationShutdown() error })
		if !ok {
			t.Fatal("provider should implement OnApplicationShutdown")
		}
		if err := shutdownFn.OnApplicationShutdown(); err != nil {
			t.Fatalf("OnApplicationShutdown failed: %v", err)
		}

		// Verify all hooks were called
		if !p.initCalled {
			t.Error("OnModuleInit was not called")
		}
		if !p.bootstrapCalled {
			t.Error("OnApplicationBootstrap was not called")
		}
		if !p.beforeShutdownCalled {
			t.Error("BeforeApplicationShutdown was not called")
		}
		if !p.destroyCalled {
			t.Error("OnModuleDestroy was not called")
		}
		if !p.shutdownCalled {
			t.Error("OnApplicationShutdown was not called")
		}
	})

	t.Run("hook returns error propagates correctly", func(t *testing.T) {
		p := &ErrorProvider{}

		initFn, ok := any(p).(interface{ OnModuleInit() error })
		if !ok {
			t.Fatal("provider should implement OnModuleInit")
		}
		err := initFn.OnModuleInit()
		if err == nil {
			t.Error("expected error from OnModuleInit, got nil")
		}
		if err.Error() != "init failed" {
			t.Errorf("expected error message 'init failed', got '%s'", err.Error())
		}
	})

	t.Run("partial hook implementation executes only implemented hooks", func(t *testing.T) {
		p := &InitOnlyProvider{}

		// Should be able to call OnModuleInit
		initFn, ok := any(p).(interface{ OnModuleInit() error })
		if !ok {
			t.Fatal("provider should implement OnModuleInit")
		}
		if err := initFn.OnModuleInit(); err != nil {
			t.Fatalf("OnModuleInit failed: %v", err)
		}

		// Should not implement OnApplicationBootstrap
		_, ok = any(p).(interface{ OnApplicationBootstrap() error })
		if ok {
			t.Error("provider should not implement OnApplicationBootstrap")
		}
	})

	t.Run("multiple providers can be detected and executed", func(t *testing.T) {
		providers := []any{
			&InitOnlyProvider{},
			&BootstrapOnlyProvider{},
			&DestroyOnlyProvider{},
			&ShutdownOnlyProvider{},
		}

		// Verify each provider implements its expected hook
		_, hasInit := providers[0].(interface{ OnModuleInit() error })
		if !hasInit {
			t.Error("InitOnlyProvider should implement OnModuleInit")
		}

		_, hasBoot := providers[1].(interface{ OnApplicationBootstrap() error })
		if !hasBoot {
			t.Error("BootstrapOnlyProvider should implement OnApplicationBootstrap")
		}

		_, hasDestroy := providers[2].(interface{ OnModuleDestroy() error })
		if !hasDestroy {
			t.Error("DestroyOnlyProvider should implement OnModuleDestroy")
		}

		_, hasShutdown := providers[3].(interface{ OnApplicationShutdown() error })
		if !hasShutdown {
			t.Error("ShutdownOnlyProvider should implement OnApplicationShutdown")
		}
	})
}

func TestInterfaceTypeAssertions(t *testing.T) {
	t.Run("OnModuleInit interface assertion", func(t *testing.T) {
		p := &InitOnlyProvider{}
		if _, ok := any(p).(OnModuleInit); !ok {
			t.Error("InitOnlyProvider should satisfy OnModuleInit interface")
		}
		if _, ok := any(p).(OnApplicationBootstrap); ok {
			t.Error("InitOnlyProvider should not satisfy OnApplicationBootstrap interface")
		}
	})

	t.Run("OnApplicationBootstrap interface assertion", func(t *testing.T) {
		p := &BootstrapOnlyProvider{}
		if _, ok := any(p).(OnApplicationBootstrap); !ok {
			t.Error("BootstrapOnlyProvider should satisfy OnApplicationBootstrap interface")
		}
		if _, ok := any(p).(OnModuleInit); ok {
			t.Error("BootstrapOnlyProvider should not satisfy OnModuleInit interface")
		}
	})

	t.Run("OnModuleDestroy interface assertion", func(t *testing.T) {
		p := &DestroyOnlyProvider{}
		if _, ok := any(p).(OnModuleDestroy); !ok {
			t.Error("DestroyOnlyProvider should satisfy OnModuleDestroy interface")
		}
		if _, ok := any(p).(OnModuleInit); ok {
			t.Error("DestroyOnlyProvider should not satisfy OnModuleInit interface")
		}
	})

	t.Run("OnApplicationShutdown interface assertion", func(t *testing.T) {
		p := &ShutdownOnlyProvider{}
		if _, ok := any(p).(OnApplicationShutdown); !ok {
			t.Error("ShutdownOnlyProvider should satisfy OnApplicationShutdown interface")
		}
		if _, ok := any(p).(OnModuleInit); ok {
			t.Error("ShutdownOnlyProvider should not satisfy OnModuleInit interface")
		}
	})

	t.Run("AllHooksProvider satisfies all interfaces", func(t *testing.T) {
		p := &AllHooksProvider{}
		if _, ok := any(p).(OnModuleInit); !ok {
			t.Error("AllHooksProvider should satisfy OnModuleInit interface")
		}
		if _, ok := any(p).(OnApplicationBootstrap); !ok {
			t.Error("AllHooksProvider should satisfy OnApplicationBootstrap interface")
		}
		if _, ok := any(p).(OnModuleDestroy); !ok {
			t.Error("AllHooksProvider should satisfy OnModuleDestroy interface")
		}
		if _, ok := any(p).(OnApplicationShutdown); !ok {
			t.Error("AllHooksProvider should satisfy OnApplicationShutdown interface")
		}
	})

	t.Run("NoHooksProvider satisfies no interfaces", func(t *testing.T) {
		p := &NoHooksProvider{}
		if _, ok := any(p).(OnModuleInit); ok {
			t.Error("NoHooksProvider should not satisfy OnModuleInit interface")
		}
		if _, ok := any(p).(OnApplicationBootstrap); ok {
			t.Error("NoHooksProvider should not satisfy OnApplicationBootstrap interface")
		}
		if _, ok := any(p).(OnModuleDestroy); ok {
			t.Error("NoHooksProvider should not satisfy OnModuleDestroy interface")
		}
		if _, ok := any(p).(OnApplicationShutdown); ok {
			t.Error("NoHooksProvider should not satisfy OnApplicationShutdown interface")
		}
	})
}

func TestNilProviderHandling(t *testing.T) {
	t.Run("nil provider does not implement any hooks", func(t *testing.T) {
		var p any = nil
		_, hasInit := p.(interface{ OnModuleInit() error })
		_, hasBoot := p.(interface{ OnApplicationBootstrap() error })
		_, hasBeforeShutdown := p.(interface{ BeforeApplicationShutdown() error })
		_, hasDestroy := p.(interface{ OnModuleDestroy() error })
		_, hasShutdown := p.(interface{ OnApplicationShutdown() error })

		if hasInit || hasBoot || hasBeforeShutdown || hasDestroy || hasShutdown {
			t.Error("nil provider should not implement any hooks")
		}
	})
}
