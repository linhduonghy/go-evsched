package memory

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScheduler(t *testing.T) {
	tests := []struct {
		name     string
		timezone string
		wantErr  bool
	}{
		{
			name:     "Valid UTC Timezone",
			timezone: "UTC",
			wantErr:  false,
		},
		{
			name:     "Valid Asia/Ho_Chi_Minh Timezone",
			timezone: "Asia/Ho_Chi_Minh",
			wantErr:  false,
		},
		{
			name:     "Invalid Timezone",
			timezone: "Invalid",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewScheduler(tt.timezone)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, s)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, s)
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				err = s.Shutdown(ctx)
				assert.NoError(t, err)
			}
		})
	}
}

func TestMemoryScheduler_AddDurationJob(t *testing.T) {
	s, err := NewScheduler("UTC")
	require.NoError(t, err)
	require.NotNil(t, s)

	ch := make(chan int, 1)
	handler := func() {
		ch <- 2
	}

	err = s.AddIntervalJob(10*time.Millisecond, handler)
	assert.NoError(t, err)

	s.Start()

	select {
	case val := <-ch:
		assert.Equal(t, 2, val)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for duration job to execute")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = s.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestMemoryScheduler_AddCronJob(t *testing.T) {
	s, err := NewScheduler("UTC")
	require.NoError(t, err)
	require.NotNil(t, s)

	memSched, ok := s.(*scheduler)
	require.True(t, ok)

	executed := make(chan struct{})
	handler := func() {
		close(executed)
	}

	// Every second cron expression
	err = memSched.addCronJob("* * * * * *", handler)
	assert.NoError(t, err)

	s.Start()

	select {
	case <-executed:
		// Cron job executed successfully
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for cron job to execute")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = s.Shutdown(ctx)
	assert.NoError(t, err)
}
