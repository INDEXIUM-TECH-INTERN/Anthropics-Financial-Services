package di

import (
	"gemini-cli/internal/domain/entities"
)

// Bindings registers all service bindings with the container
func Bindings(container *Container) {
	// Register agents
	bindAgents(container)

	// Register services
	bindServices(container)

	// Register infrastructure components
	bindInfrastructure(container)

	// Register repositories
	bindRepositories(container)
}

// bindAgents registers agents
func bindAgents(container *Container) {
	// Register the default agent
	defaultAgent := entities.NewAgent(
		"financial-agent",
		"Financial Agent",
		"AI assistant specialized in financial analysis and market research",
		[]string{
			"financial-analysis",
			"market-research",
			"calculation",
			"report-generation",
		},
	)
	container.Register(defaultAgent)
}

// bindServices registers application services
func bindServices(container *Container) {
	// Register agent service interface implementation
	container.Register(&AgentServiceImpl{})

	// Register orchestrator
	container.Register(&ReActOrchestrator{})

	// Register tool registry
	container.Register(&DefaultToolRegistry{})
}

// bindInfrastructure registers infrastructure components
func bindInfrastructure(container *Container) {
	// Register LLM providers
	container.Register(&GeminiProviderImpl{})

	// Register tool executors
	container.Register(&FinancialResearchToolExecutor{})
	container.Register(&FinancialScrapeToolExecutor{})
	container.Register(&FinancialCalculateToolExecutor{})
	container.Register(&ExportReportToolExecutor{})
}

// bindRepositories registers repositories
func bindRepositories(container *Container) {
	// Register conversation repository
	container.Register(&MemoryConversationRepository{})

	// Register cache repository
	container.Register(&MemoryCacheRepository{})
}