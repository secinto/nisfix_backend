package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/checkfix-tools/nisfix_backend/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Seeder handles database seeding operations
// #SEED_DATA: System questionnaire templates for ISO27001, GDPR, NIS2
type Seeder struct {
	db *mongo.Database
}

// NewSeeder creates a new database seeder
func NewSeeder(db *mongo.Database) *Seeder {
	return &Seeder{db: db}
}

// SeedAll runs all seed operations
func (s *Seeder) SeedAll(ctx context.Context) error {
	log.Println("Starting database seeding...")

	if err := s.SeedQuestionnaireTemplates(ctx); err != nil {
		return fmt.Errorf("failed to seed questionnaire templates: %w", err)
	}

	log.Println("Database seeding completed successfully")
	return nil
}

// SeedQuestionnaireTemplates seeds the system questionnaire templates
// #SEED_DATA: ISO27001, GDPR, NIS2 templates with topics
func (s *Seeder) SeedQuestionnaireTemplates(ctx context.Context) error {
	collection := s.db.Collection(models.QuestionnaireTemplate{}.CollectionName())

	// Check if templates already exist
	count, err := collection.CountDocuments(ctx, bson.M{"is_system": true})
	if err != nil {
		return err
	}
	if count > 0 {
		log.Println("System templates already exist, skipping seeding")
		return nil
	}

	templates := s.getSystemTemplates()

	// Convert to interface slice for InsertMany
	docs := make([]interface{}, len(templates))
	for i, t := range templates {
		t.BeforeCreate()
		t.Publish()
		docs[i] = t
	}

	_, err = collection.InsertMany(ctx, docs)
	if err != nil {
		return err
	}

	log.Printf("Seeded %d system questionnaire templates", len(templates))
	return nil
}

// getSystemTemplates returns the system templates for ISO27001, GDPR, and NIS2
func (s *Seeder) getSystemTemplates() []*models.QuestionnaireTemplate {
	now := time.Now().UTC()

	return []*models.QuestionnaireTemplate{
		// ISO 27001 Basic Assessment
		{
			ID:                  primitive.NewObjectID(),
			Name:                "ISO 27001 Basic Assessment",
			Description:         "A comprehensive assessment based on ISO/IEC 27001 information security management system requirements. Covers essential security controls and practices.",
			Category:            models.TemplateCategoryISO27001,
			Version:             "1.0",
			IsSystem:            true,
			DefaultPassingScore: 70,
			EstimatedMinutes:    45,
			Topics: []models.TemplateTopic{
				{
					ID:          "info-security-policy",
					Name:        "Information Security Policy",
					Description: "Organizational policies and procedures for information security management",
					Order:       1,
				},
				{
					ID:          "access-control",
					Name:        "Access Control",
					Description: "User access management, authentication, and authorization controls",
					Order:       2,
				},
				{
					ID:          "cryptography",
					Name:        "Cryptography",
					Description: "Use of cryptographic controls to protect information",
					Order:       3,
				},
				{
					ID:          "physical-security",
					Name:        "Physical Security",
					Description: "Physical and environmental security measures",
					Order:       4,
				},
				{
					ID:          "operations-security",
					Name:        "Operations Security",
					Description: "Operational procedures and responsibilities",
					Order:       5,
				},
				{
					ID:          "incident-management",
					Name:        "Incident Management",
					Description: "Security incident management and response",
					Order:       6,
				},
				{
					ID:          "business-continuity",
					Name:        "Business Continuity",
					Description: "Business continuity planning and disaster recovery",
					Order:       7,
				},
				{
					ID:          "compliance",
					Name:        "Compliance",
					Description: "Compliance with legal and regulatory requirements",
					Order:       8,
				},
			},
			Tags:       []string{"iso27001", "information-security", "isms", "security-management"},
			UsageCount: 0,
			CreatedAt:  now,
			UpdatedAt:  now,
		},

		// ISO 27001 Comprehensive Assessment
		{
			ID:                  primitive.NewObjectID(),
			Name:                "ISO 27001 Comprehensive Assessment",
			Description:         "An in-depth assessment covering all 114 controls of ISO/IEC 27001:2013 Annex A. Suitable for detailed security evaluations.",
			Category:            models.TemplateCategoryISO27001,
			Version:             "1.0",
			IsSystem:            true,
			DefaultPassingScore: 75,
			EstimatedMinutes:    90,
			Topics: []models.TemplateTopic{
				{
					ID:          "a5-policies",
					Name:        "A.5 Information Security Policies",
					Description: "Management direction for information security",
					Order:       1,
				},
				{
					ID:          "a6-organization",
					Name:        "A.6 Organization of Information Security",
					Description: "Internal organization and mobile devices/teleworking",
					Order:       2,
				},
				{
					ID:          "a7-hr-security",
					Name:        "A.7 Human Resource Security",
					Description: "Prior to, during, and termination of employment",
					Order:       3,
				},
				{
					ID:          "a8-asset-management",
					Name:        "A.8 Asset Management",
					Description: "Responsibility for assets and information classification",
					Order:       4,
				},
				{
					ID:          "a9-access-control",
					Name:        "A.9 Access Control",
					Description: "Business requirements, user access, and system access",
					Order:       5,
				},
				{
					ID:          "a10-cryptography",
					Name:        "A.10 Cryptography",
					Description: "Cryptographic controls",
					Order:       6,
				},
				{
					ID:          "a11-physical",
					Name:        "A.11 Physical and Environmental Security",
					Description: "Secure areas and equipment security",
					Order:       7,
				},
				{
					ID:          "a12-operations",
					Name:        "A.12 Operations Security",
					Description: "Operational procedures and protection from malware",
					Order:       8,
				},
				{
					ID:          "a13-communications",
					Name:        "A.13 Communications Security",
					Description: "Network security management and information transfer",
					Order:       9,
				},
				{
					ID:          "a14-development",
					Name:        "A.14 System Development",
					Description: "Security requirements and development processes",
					Order:       10,
				},
				{
					ID:          "a15-supplier",
					Name:        "A.15 Supplier Relationships",
					Description: "Security in supplier agreements and delivery",
					Order:       11,
				},
				{
					ID:          "a16-incident",
					Name:        "A.16 Incident Management",
					Description: "Management of information security incidents",
					Order:       12,
				},
				{
					ID:          "a17-continuity",
					Name:        "A.17 Business Continuity",
					Description: "Information security continuity and redundancies",
					Order:       13,
				},
				{
					ID:          "a18-compliance",
					Name:        "A.18 Compliance",
					Description: "Compliance with legal and contractual requirements",
					Order:       14,
				},
			},
			Tags:       []string{"iso27001", "comprehensive", "annex-a", "security-controls"},
			UsageCount: 0,
			CreatedAt:  now,
			UpdatedAt:  now,
		},

		// GDPR Compliance Checklist
		{
			ID:                  primitive.NewObjectID(),
			Name:                "GDPR Compliance Checklist",
			Description:         "Assessment of General Data Protection Regulation (GDPR) compliance requirements. Covers data processing, consent, and data subject rights.",
			Category:            models.TemplateCategoryGDPR,
			Version:             "1.0",
			IsSystem:            true,
			DefaultPassingScore: 80,
			EstimatedMinutes:    60,
			Topics: []models.TemplateTopic{
				{
					ID:          "lawful-basis",
					Name:        "Lawful Basis for Processing",
					Description: "Legal grounds for processing personal data",
					Order:       1,
				},
				{
					ID:          "consent-management",
					Name:        "Consent Management",
					Description: "Obtaining, recording, and managing consent",
					Order:       2,
				},
				{
					ID:          "data-subject-rights",
					Name:        "Data Subject Rights",
					Description: "Handling rights requests (access, rectification, erasure, etc.)",
					Order:       3,
				},
				{
					ID:          "privacy-notices",
					Name:        "Privacy Notices",
					Description: "Transparency and privacy information requirements",
					Order:       4,
				},
				{
					ID:          "data-protection-officer",
					Name:        "Data Protection Officer",
					Description: "DPO appointment and responsibilities",
					Order:       5,
				},
				{
					ID:          "data-processing-agreements",
					Name:        "Data Processing Agreements",
					Description: "Contracts with data processors and controllers",
					Order:       6,
				},
				{
					ID:          "international-transfers",
					Name:        "International Transfers",
					Description: "Cross-border data transfer mechanisms",
					Order:       7,
				},
				{
					ID:          "breach-notification",
					Name:        "Breach Notification",
					Description: "Data breach detection and notification procedures",
					Order:       8,
				},
				{
					ID:          "privacy-by-design",
					Name:        "Privacy by Design",
					Description: "Data protection by design and by default",
					Order:       9,
				},
				{
					ID:          "records-processing",
					Name:        "Records of Processing",
					Description: "Documentation of processing activities",
					Order:       10,
				},
				{
					ID:          "dpia",
					Name:        "Data Protection Impact Assessment",
					Description: "DPIA processes and implementation",
					Order:       11,
				},
			},
			Tags:       []string{"gdpr", "data-protection", "privacy", "eu-regulation"},
			UsageCount: 0,
			CreatedAt:  now,
			UpdatedAt:  now,
		},

		// GDPR Quick Assessment
		{
			ID:                  primitive.NewObjectID(),
			Name:                "GDPR Quick Assessment",
			Description:         "A streamlined GDPR compliance check focusing on the most critical requirements. Ideal for initial assessments.",
			Category:            models.TemplateCategoryGDPR,
			Version:             "1.0",
			IsSystem:            true,
			DefaultPassingScore: 75,
			EstimatedMinutes:    30,
			Topics: []models.TemplateTopic{
				{
					ID:          "data-processing",
					Name:        "Data Processing",
					Description: "Lawful basis and processing principles",
					Order:       1,
				},
				{
					ID:          "consent",
					Name:        "Consent",
					Description: "Consent collection and management",
					Order:       2,
				},
				{
					ID:          "individual-rights",
					Name:        "Individual Rights",
					Description: "Supporting data subject rights",
					Order:       3,
				},
				{
					ID:          "security-measures",
					Name:        "Security Measures",
					Description: "Technical and organizational measures",
					Order:       4,
				},
				{
					ID:          "third-party",
					Name:        "Third Party Management",
					Description: "Processor and sub-processor management",
					Order:       5,
				},
			},
			Tags:       []string{"gdpr", "quick-check", "essential", "compliance"},
			UsageCount: 0,
			CreatedAt:  now,
			UpdatedAt:  now,
		},

		// NIS2 Security Assessment
		{
			ID:                  primitive.NewObjectID(),
			Name:                "NIS2 Security Assessment",
			Description:         "Assessment based on the NIS2 Directive (EU 2022/2555) requirements for essential and important entities. Covers cybersecurity risk management measures.",
			Category:            models.TemplateCategoryNIS2,
			Version:             "1.0",
			IsSystem:            true,
			DefaultPassingScore: 75,
			EstimatedMinutes:    75,
			Topics: []models.TemplateTopic{
				{
					ID:          "governance",
					Name:        "Governance and Accountability",
					Description: "Management body responsibilities and cybersecurity governance",
					Order:       1,
				},
				{
					ID:          "risk-management",
					Name:        "Risk Management",
					Description: "Risk analysis policies and procedures",
					Order:       2,
				},
				{
					ID:          "incident-handling",
					Name:        "Incident Handling",
					Description: "Incident detection, response, and reporting",
					Order:       3,
				},
				{
					ID:          "business-continuity",
					Name:        "Business Continuity",
					Description: "Backup management, disaster recovery, and crisis management",
					Order:       4,
				},
				{
					ID:          "supply-chain",
					Name:        "Supply Chain Security",
					Description: "Security in supplier and service provider relationships",
					Order:       5,
				},
				{
					ID:          "network-security",
					Name:        "Network and Systems Security",
					Description: "Security in network and information systems acquisition and development",
					Order:       6,
				},
				{
					ID:          "vulnerability-management",
					Name:        "Vulnerability Management",
					Description: "Vulnerability handling and disclosure",
					Order:       7,
				},
				{
					ID:          "access-management",
					Name:        "Access Management",
					Description: "Access control policies and multi-factor authentication",
					Order:       8,
				},
				{
					ID:          "cryptography",
					Name:        "Cryptography",
					Description: "Use of cryptography and encryption",
					Order:       9,
				},
				{
					ID:          "hr-security",
					Name:        "Human Resources Security",
					Description: "Employee security awareness and training",
					Order:       10,
				},
				{
					ID:          "asset-management",
					Name:        "Asset Management",
					Description: "Asset inventory and classification",
					Order:       11,
				},
				{
					ID:          "effectiveness",
					Name:        "Effectiveness Assessment",
					Description: "Policies and procedures for assessing effectiveness",
					Order:       12,
				},
			},
			Tags:       []string{"nis2", "cybersecurity", "eu-directive", "critical-infrastructure"},
			UsageCount: 0,
			CreatedAt:  now,
			UpdatedAt:  now,
		},

		// NIS2 Quick Readiness Check
		{
			ID:                  primitive.NewObjectID(),
			Name:                "NIS2 Quick Readiness Check",
			Description:         "A preliminary assessment to evaluate NIS2 readiness. Helps identify major gaps before a comprehensive assessment.",
			Category:            models.TemplateCategoryNIS2,
			Version:             "1.0",
			IsSystem:            true,
			DefaultPassingScore: 70,
			EstimatedMinutes:    30,
			Topics: []models.TemplateTopic{
				{
					ID:          "scope-classification",
					Name:        "Scope and Classification",
					Description: "Entity classification and NIS2 applicability",
					Order:       1,
				},
				{
					ID:          "basic-security",
					Name:        "Basic Security Measures",
					Description: "Fundamental security controls in place",
					Order:       2,
				},
				{
					ID:          "incident-reporting",
					Name:        "Incident Reporting",
					Description: "Capability to report incidents within required timeframes",
					Order:       3,
				},
				{
					ID:          "supply-chain-overview",
					Name:        "Supply Chain Overview",
					Description: "Basic supply chain security measures",
					Order:       4,
				},
				{
					ID:          "management-awareness",
					Name:        "Management Awareness",
					Description: "Management body awareness and training",
					Order:       5,
				},
			},
			Tags:       []string{"nis2", "readiness", "quick-check", "preliminary"},
			UsageCount: 0,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}
}

// ClearSeededData removes all seeded system templates
func (s *Seeder) ClearSeededData(ctx context.Context) error {
	collection := s.db.Collection(models.QuestionnaireTemplate{}.CollectionName())

	result, err := collection.DeleteMany(ctx, bson.M{"is_system": true})
	if err != nil {
		return err
	}

	log.Printf("Removed %d system templates", result.DeletedCount)
	return nil
}
