### **ZAMM: A Specification-Driven Literate Programming Tool**

#### **1. Executive Summary**

This document outlines the development of a literate programming tool that tracks the relationship between human-authored requirements and machine-authored code. LLM-generated code often comes with a decent amount of cruft; as such, this tool aims to preserve the human vision for the project inside of a natural language repository with a high SNR. The system is designed for a collaborative workflow between human developers, who define and structure requirements, and LLM assistants, which generate and refactor code based on those specifications. The primary goals are to allow for the easy reimplementation of software across different stacks, and the easy maintenance of forked software regardless of major upstream changes. The primary strategy of achieving these goals is by maintaining a clear and evolving record of project requirements, automating implementation and reimplementation tasks, and preserving the context of architectural decisions as well as the implications of how these decisions interact with individual features.

---

#### **2. Core Concepts**

The tool is built on two primary hierarchical structures: **Specifications** (the "what") and **Implementations/Scopes** (the "how").

##### **2.1. Specifications: The Source of Truth**
The specification hierarchy is the canonical repository for all high-signal, human-authored project requirements.

*   **Structure:** A hierarchical tree of "Spec Nodes."
*   **User Experience:** Users can start with a single root node (e.g., "New Python Script") and progressively add, nest, split, merge, and refactor nodes as requirements evolve.
*   **Content:** Free-form Markdown is supported for readability, augmented with optional structured fields for machine parsing.
*   **Exclusion of Noise:** LLM-generated code is explicitly *not* the source of truth; it is a derivative product of the human-written specs.

##### **2.2. Implementations & Scopes: The Architectural Blueprint**
The implementation structure captures the concrete realization of the specifications, organized by architectural choices.

*   **Structure:** A Directed Acyclic Graph (DAG) of "Scope Nodes."
*   **Architectural Forks:** Each branch in the DAG represents a major design or architectural decision. For example, a "Python" scope could fork into "Python with Flask" and "Python with Django" child scopes.
*   **Inheritance:** Scopes form an inheritance hierarchy. When generating code for a specific scope (a leaf node), the LLM is provided with the full context of all its ancestor scopes. Child scope attributes and directives override those from a parent.
*   **User Control:** New scopes, representing a project fork, are defined by users. These are intended to be lightweight and created for significant architectural divergence.

---

#### **3. The Node System: Atomic Units of Information**

Nodes are the fundamental building blocks of both the specification and implementation hierarchies.

##### **3.1. General Node Properties**
*   **Content Format:** Free-form Markdown with optional, structured key-value fields.
*   **Core Fields:**
    *   `node_type`: A mandatory field indicating whether it is a `spec` or `impl` node.
    *   `context_links`: A list of other nodes that must be included to provide full context.

##### **3.2. Specification Nodes**
*   **Atomic Unit:** Represents a single, specific requirement for functionality, testing, or infrastructure.
*   **Identification & Versioning:**
    *   **Stable ID:** A permanent, unique identifier to track the evolution of a requirement over time.
    *   **Version-Specific ID:** A unique identifier for each version of a spec node.
*   **Relationships & Linking:**
    *   Spec nodes can reference any other spec node to establish a dependency or provide context.
    *   When a reference is created, the system records both the **stable ID** of the target node and the **specific version ID** being referenced at that moment. By default, links point to the latest version.
    *   A generic relationship type will be supported to simply denote that one node's information is necessary for understanding another.
*   **Categorization Flags:**
    *   **Implementation Specificity:** A flag (or color-coding) to distinguish between:
        *   **Universal Specs:** Requirements applicable to all implementations. This would generally refer to user-visible functionality, such as "User should be able to login via Telegram."
        *   **Implementation-Specific Specs:** Requirements that apply only to a particular scope (e.g., Python-specific details hidden from a JavaScript implementation). This would generally refer to implementation details, such as "The login handler class should be named `TelegramLogin`." Because this will always reflect human-authored desires, it is still assumed to be higher signal data than the human editing of LLM-generated code.
    *   **Behavior Type:** A flag to classify the requirement as relating to:
        *   `runtime_behavior`
        *   `test_behavior`
        *   `infrastructure_behavior`

##### **3.3. Implementation (Scope) Nodes**
*   **Content:** Contains links to reference implementations, such as specific code samples or commits.
*   **LLM Directives:** Supports explicit, documented directives for the LLM, such as code generation style, architectural patterns, or project structure information for that specific implementation.

---

#### **4. LLM-Assisted Workflow and Automation**

The tool's primary function is to orchestrate the interaction between human-defined specs and LLM-driven code generation.

##### **4.1. Code Generation and Reimplementation**
*   **Process:** The tool supports automated code generation and reimplementation (e.g., creating a Rust version from a Python reference) with a human-in-the-loop for review.
*   **Context Gathering:** Before invoking the LLM, the system automatically gathers all relevant spec nodes and existing code samples from parent/sibling implementations. This context is optionally presented to the user for review and confirmation.
*   **Intelligent System Prompts:** The tool leverages its knowledge of the project structure and scope hierarchy to construct highly contextualized system prompts for the LLM, improving the quality of generated code.

##### **4.2. Linking Specs, Code, and Provenance**
*   **Spec-to-Code Links:** Spec nodes are linked to implementation nodes via objects representing specific code samples or commits.
*   **Code Sample Uniqueness:** A code sample is strictly linked to a single scope. Even if a reimplementation produces identical code, it is stored as a new, distinct object within its own scope.
*   **Provenance Tracking:** If one code sample is used as a basis for generating another, this relationship (provenance) is explicitly documented.

##### **4.3. Automated Validation and Change Management**
*   **Automated Checks:** The system will automatically attempt to compile and run tests on generated code. It will retry a configurable number of times (N) before flagging the code for mandatory human review.
*   **Status Tracking:** Code samples will have a human-reviewed status flag and a separate automated-check status flag (pass/fail).
*   **Staleness Notification:** The UI will clearly inform users when a code sample is outdated because its underlying spec has been modified.
*   **Dependency Analysis:** After a spec node is changed, the tool will query the LLM to assess if dependent specs require re-validation. If so, the user will be prompted to review them.

---

#### **5. Operational and Environmental Tooling**

The tool will assist with the practical aspects of the development lifecycle.

*   **Common Operations:** The system will keep a documented record of common operational procedures, both for the codebase (e.g., "how to add a new user settings page") and the project itself (e.g., "how to build and deploy").
*   **Environment Setup:** The LLM will be tasked with generating and executing command-line instructions to install necessary development tools and dependencies.
*   **Logging:** All setup steps, user assistance provided during setup, and any resulting error logs will be meticulously recorded for reproducibility and debugging.