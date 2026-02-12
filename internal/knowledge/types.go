package knowledge

// ThreatPattern represents a complete threat pattern from the knowledge base
type ThreatPattern struct {
	ID          string `yaml:"id" json:"id"`
	Name        string `yaml:"name" json:"name"`
	Tier        string `yaml:"tier" json:"tier"`
	Version     string `yaml:"version" json:"version"`
	LastUpdated string `yaml:"last_updated" json:"last_updated"`

	// Scope tags
	Category    string `yaml:"category" json:"category"`
	Subcategory string `yaml:"subcategory" json:"subcategory"`
	Language    string `yaml:"language" json:"language"`
	Framework   string `yaml:"framework" json:"framework"`

	Severity   string `yaml:"severity" json:"severity"`
	Likelihood string `yaml:"likelihood" json:"likelihood"`

	// Generalization
	GeneralizesTo []string `yaml:"generalizes_to,omitempty" json:"generalizes_to,omitempty"`

	// Provenance
	Provenance Provenance `yaml:"provenance" json:"provenance"`

	// Triggers for agent matching
	Triggers Triggers `yaml:"triggers" json:"triggers"`

	// Differentiation from LLM knowledge
	Differentiation Differentiation `yaml:"differentiation" json:"differentiation"`

	// Core content
	Description  string       `yaml:"description" json:"description"`
	AgentSummary AgentSummary `yaml:"agent_summary" json:"agent_summary"`

	// Attack scenario (Tier A only)
	AttackScenario *AttackScenario `yaml:"attack_scenario,omitempty" json:"attack_scenario,omitempty"`

	// Mitigations
	Mitigations []Mitigation `yaml:"mitigations" json:"mitigations"`

	// Security principles (Tier A only)
	SecurityPrinciples []SecurityPrinciple `yaml:"security_principles,omitempty" json:"security_principles,omitempty"`

	// Related patterns
	RelatedPatterns []RelatedPattern `yaml:"related_patterns,omitempty" json:"related_patterns,omitempty"`

	// Testing guidance (Tier A only)
	Testing *Testing `yaml:"testing,omitempty" json:"testing,omitempty"`

	// Validation results
	Validation *Validation `yaml:"validation,omitempty" json:"validation,omitempty"`
}

// Provenance tracks the source of the threat pattern
type Provenance struct {
	SourceType       string            `yaml:"source_type" json:"source_type"`
	Description      string            `yaml:"description" json:"description"`
	PublicReferences []PublicReference `yaml:"public_references,omitempty" json:"public_references,omitempty"`
}

// PublicReference links to external security resources
type PublicReference struct {
	CWE   string `yaml:"cwe,omitempty" json:"cwe,omitempty"`
	OWASP string `yaml:"owasp,omitempty" json:"owasp,omitempty"`
	Name  string `yaml:"name,omitempty" json:"name,omitempty"`
	URL   string `yaml:"url,omitempty" json:"url,omitempty"`
}

// Triggers define when an agent should query this pattern
type Triggers struct {
	Keywords     []string `yaml:"keywords" json:"keywords"`
	Actions      []string `yaml:"actions" json:"actions"`
	FilePatterns []string `yaml:"file_patterns" json:"file_patterns"`
}

// Differentiation explains why TMKB adds value beyond LLM knowledge
type Differentiation struct {
	LLMKnowledgeState string   `yaml:"llm_knowledge_state" json:"llm_knowledge_state"`
	TMKBValue         string   `yaml:"tmkb_value" json:"tmkb_value"`
	LLMBlindspots     []string `yaml:"llm_blindspots" json:"llm_blindspots"`
}

// AgentSummary is the concise output for AI agents (<100 tokens)
type AgentSummary struct {
	Threat string `yaml:"threat" json:"threat"`
	Check  string `yaml:"check" json:"check"`
	Fix    string `yaml:"fix" json:"fix"`
}

// AttackScenario describes how the vulnerability is exploited
type AttackScenario struct {
	Narrative     string       `yaml:"narrative" json:"narrative"`
	Preconditions []string     `yaml:"preconditions" json:"preconditions"`
	AttackSteps   []AttackStep `yaml:"attack_steps" json:"attack_steps"`
	Impact        Impact       `yaml:"impact" json:"impact"`
}

// AttackStep is a single step in an attack scenario
type AttackStep struct {
	Step   int    `yaml:"step" json:"step"`
	Action string `yaml:"action" json:"action"`
	Detail string `yaml:"detail" json:"detail"`
}

// Impact describes the security impact of the vulnerability
type Impact struct {
	Confidentiality string `yaml:"confidentiality" json:"confidentiality"`
	Integrity       string `yaml:"integrity" json:"integrity"`
	Availability    string `yaml:"availability" json:"availability"`
	Scope           string `yaml:"scope" json:"scope"`
	BusinessImpact  string `yaml:"business_impact,omitempty" json:"business_impact,omitempty"`
}

// Mitigation describes how to fix or prevent the vulnerability
type Mitigation struct {
	ID                   string        `yaml:"id" json:"id"`
	Name                 string        `yaml:"name,omitempty" json:"name,omitempty"`
	Description          string        `yaml:"description" json:"description"`
	Effectiveness        string        `yaml:"effectiveness" json:"effectiveness"`
	ImplementationEffort string        `yaml:"implementation_effort" json:"implementation_effort"`
	Tradeoffs            []string      `yaml:"tradeoffs,omitempty" json:"tradeoffs,omitempty"`
	CodeExamples         []CodeExample `yaml:"code_examples,omitempty" json:"code_examples,omitempty"`
}

// CodeExample shows vulnerable and/or secure code
type CodeExample struct {
	Language       string `yaml:"language" json:"language"`
	Framework      string `yaml:"framework" json:"framework"`
	Description    string `yaml:"description" json:"description"`
	VulnerableCode string `yaml:"vulnerable_code,omitempty" json:"vulnerable_code,omitempty"`
	SecureCode     string `yaml:"secure_code,omitempty" json:"secure_code,omitempty"`
}

// SecurityPrinciple is a general security principle illustrated by this pattern
type SecurityPrinciple struct {
	Principle   string `yaml:"principle" json:"principle"`
	Explanation string `yaml:"explanation" json:"explanation"`
}

// RelatedPattern links to other patterns in the knowledge base
type RelatedPattern struct {
	ID           string `yaml:"id" json:"id"`
	Relationship string `yaml:"relationship" json:"relationship"`
	Description  string `yaml:"description" json:"description"`
}

// Testing provides guidance for verifying the pattern
type Testing struct {
	ManualVerification []ManualCheck    `yaml:"manual_verification,omitempty" json:"manual_verification,omitempty"`
	AutomatedChecks    []AutomatedCheck `yaml:"automated_checks,omitempty" json:"automated_checks,omitempty"`
}

// ManualCheck is a step for manual verification
type ManualCheck struct {
	Step  string `yaml:"step" json:"step"`
	Check string `yaml:"check" json:"check"`
}

// AutomatedCheck describes an automated test
type AutomatedCheck struct {
	Type        string `yaml:"type" json:"type"`
	Description string `yaml:"description" json:"description"`
	Pattern     string `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	Expectation string `yaml:"expectation,omitempty" json:"expectation,omitempty"`
}

// Validation records baseline test results
type Validation struct {
	BaselineTest *BaselineTest `yaml:"baseline_test,omitempty" json:"baseline_test,omitempty"`
}

// BaselineTest documents LLM behavior without TMKB
type BaselineTest struct {
	Prompt          string `yaml:"prompt" json:"prompt"`
	ExpectedFailure string `yaml:"expected_failure" json:"expected_failure"`
	Observed        string `yaml:"observed" json:"observed"`
	Date            string `yaml:"date" json:"date"`
}
