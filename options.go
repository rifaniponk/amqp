package amqp

import (
	"context"
	"fmt"
	"time"

	"github.com/devimteam/amqp/conn"
	"github.com/devimteam/amqp/logger"
	"github.com/satori/go.uuid"
	"github.com/streadway/amqp"
)

type (
	// Options is a struct with almost all possible options of Client.
	options struct {
		wait struct {
			flag    bool
			timeout time.Duration
		}
		timeout struct {
			base time.Duration
			cap  int
		}
		subEventChanBuffer int
		log                struct {
			debug logger.Logger
			info  logger.Logger
			warn  logger.Logger
			error logger.Logger
		}
		context              context.Context
		msgOpts              messageOptions
		processAllDeliveries bool
		handlersAmount       int
		errorBefore          []ErrorBefore
		lazyCommands         bool
		connOpts             []conn.ConnectionOption
	}

	MessageIdBuilder func() string                                         // Function, that should return new message Id.
	Typer            func(value interface{}) string                        // Function, that should return string representation of type's value.
	PublishingBefore func(context.Context, *amqp.Publishing)               // Function, that changes message before publishing.
	DeliveryBefore   func(context.Context, *amqp.Delivery) context.Context // Function, that changes message before delivering.
	ErrorBefore      func(amqp.Delivery, error) error                      // Function, that changes error, which caused on incorrect handling.
)

const (
	MaxMessagePriority = 9
	MinMessagePriority = 0
)

const (
	defaultWaitDeadline  = time.Second * 5
	defaultEventBuffer   = 1
	defaultHandlerAmount = 1
)

func defaultOptions() options {
	opts := options{}
	opts.context = context.Background()
	opts.wait.timeout = defaultWaitDeadline
	opts.subEventChanBuffer = defaultEventBuffer
	opts.msgOpts.idBuilder = noopMessageIdBuilder
	opts.msgOpts.minPriority = MinMessagePriority
	opts.msgOpts.maxPriority = MaxMessagePriority
	opts.msgOpts.typer = noopTyper
	opts.handlersAmount = defaultHandlerAmount
	opts.log.debug = logger.NoopLogger
	opts.log.info = logger.NoopLogger
	opts.log.warn = logger.NoopLogger
	opts.log.error = logger.NoopLogger
	opts.msgOpts.defaultContentType = "application/json"
	return opts
}

type Option func(*options)

// WaitConnection tells client to wait connection before Sub or Pub executing.
func WaitConnection(should bool, timeout time.Duration) Option {
	return func(options *options) {
		options.wait.flag = should
		if timeout != 0 {
			options.wait.timeout = timeout
		}
	}
}

// EventChanBuffer sets the buffer of event channel for Sub method.
func EventChanBuffer(a int) Option {
	return func(options *options) {
		options.subEventChanBuffer = a
	}
}

// Context sets root context of Sub method for each event.
// context.Background by default.
func Context(ctx context.Context) Option {
	return func(options *options) {
		options.context = ctx
	}
}

// SetMessageIdBuilder sets function, that executes on every Pub call and result can be interpreted as message Id.
func SetMessageIdBuilder(builder MessageIdBuilder) Option {
	return func(options *options) {
		options.msgOpts.idBuilder = builder
	}
}

// AllowedPriority rejects messages, which not in range.
func AllowedPriority(from, to uint8) Option {
	return func(options *options) {
		options.msgOpts.minPriority = from
		options.msgOpts.maxPriority = to
	}
}

// ApplicationId adds this Id to each message, which created on Pub call.
func ApplicationId(id string) Option {
	return func(options *options) {
		options.msgOpts.applicationId = id
	}
}

// ApplicationId adds this Id to each message, which created on Pub call.
func UserId(id string) Option {
	return func(options *options) {
		options.msgOpts.userId = id
	}
}

// InfoLogger option sets logger, which logs info messages.
func InfoLogger(lg logger.Logger) Option {
	return func(options *options) {
		options.log.info = lg
	}
}

// DebugLogger option sets logger, which logs debug messages.
func DebugLogger(lg logger.Logger) Option {
	return func(options *options) {
		options.log.debug = lg
	}
}

// ErrorLogger option sets logger, which logs error messages.
func ErrorLogger(lg logger.Logger) Option {
	return func(options *options) {
		options.log.error = lg
	}
}

// WarnLogger option sets logger, which logs warning messages.
func WarnLogger(lg logger.Logger) Option {
	return func(options *options) {
		options.log.warn = lg
	}
}

// AllLoggers option is a shortcut for call each <*>Logger with the same logger.
func AllLoggers(lg logger.Logger) Option {
	return func(options *options) {
		options.log.info = lg
		options.log.debug = lg
		options.log.error = lg
		options.log.warn = lg
	}
}

func PublishBefore(before ...PublishingBefore) Option {
	return func(options *options) {
		for i := range before {
			options.msgOpts.pubBefore = append(options.msgOpts.pubBefore, before[i])
		}
	}
}

func DeliverBefore(before ...DeliveryBefore) Option {
	return func(options *options) {
		for i := range before {
			options.msgOpts.deliveryBefore = append(options.msgOpts.deliveryBefore, before[i])
		}
	}
}

// Add this option with true value that allows you to handle all deliveries from current channel, even if the Done was sent.
func ProcessAllDeliveries(v bool) Option {
	return func(options *options) {
		options.processAllDeliveries = v
	}
}

// HandlersAmount sets the amount of handle processes, which receive deliveries from one channel.
// For n > 1 client does not guarantee the order of events.
func HandlersAmount(n int) Option {
	return func(options *options) {
		if n > 0 {
			options.handlersAmount = n
		}
	}
}

// LazyDeclaring option with true value tells the Client not to declare exchanges and queues
// if it was declared before by this Client.
// By default client declares it on every Sub loop and every Pub call.
func LazyDeclaring(v bool) Option {
	return func(options *options) {
		options.lazyCommands = v
	}
}

// SetDefaultContentType sets content type which codec should be used if ContentType field of message is empty.
func SetDefaultContentType(t string) Option {
	return func(options *options) {
		options.msgOpts.defaultContentType = t
	}
}

var noopMessageIdBuilder = func() string {
	return ""
}

var noopTyper = func(_ interface{}) string {
	return ""
}

func CommonTyper(v interface{}) string {
	return fmt.Sprintf("%T", v)
}

func CommonMessageIdBuilder() string {
	return uuid.NewV4().String()
}
