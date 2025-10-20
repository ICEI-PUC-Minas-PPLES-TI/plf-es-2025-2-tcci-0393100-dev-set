# SET CLI - Demo Video Script

**Duration**: ~5-7 minutes
**Goal**: Showcase all key features of SET CLI with the Modern Minimal UI

---

## Pre-Recording Setup

### 1. Verify Example Files
```bash
# Navigate to project directory
cd "C:\Users\Inácio Moraes\Documents\GitHub\plf-es-2025-2-tcci-0393100-dev-set\Codigo"

# Verify example files exist
dir examples\batch_example.json
dir examples\batch_example.csv

# Verify binary exists
dir bin\set.exe
```

**Note**: The `examples` folder contains ready-to-use sample data:
- `batch_example.json` - Sprint 15 backlog with 5 realistic tasks (OAuth, bug fix, PDF export, refactoring, testing)
- `batch_example.csv` - Same tasks in CSV format for spreadsheet compatibility

### 2. Terminal Setup
```bash
# Clear terminal
cls

# Set terminal to full screen or large window
# Font size: 14-16pt recommended
```

### 3. Clean Slate (Optional)
```bash
# If you want to start fresh, delete existing data
del "%USERPROFILE%\.set\data.db"
```

### 4. Final Checklist
- ✅ Terminal font size: 14-16pt for visibility
- ✅ Use full screen or large window
- ✅ Ensure terminal supports UTF-8 characters
- ✅ OpenAI API key configured in `.set.yaml`
- ✅ Example files present in `examples/` folder

---

## Video Script

### INTRO (15 seconds)

**Script**:
> "Hi! Today I'm showing you SET CLI - a Software Estimation Tool that uses AI and historical data to provide accurate task estimates. Let's dive in!"

**Action**: Show terminal with clear screen

---

### PART 1: Getting Started (45 seconds)

#### Step 1: Show Help
**Script**: "First, let's see what SET can do."

```bash
./bin/set.exe --help
```

**Pause**: Let viewers see the features list (2-3 seconds)

**Script**: "SET supports AI-powered estimation, historical data analysis, batch processing, and more."

---

#### Step 2: Check Version
**Script**: "Let's check the version."

```bash
./bin/set.exe version
```

---

### PART 2: Loading Sample Data (30 seconds)

#### Step 3: Seed Database
**Script**: "SET needs historical data for better estimates. Let's load our sample dataset from the SiP research project - 1200 real software tasks."

```bash
./bin/set.exe dev seed --count 50 --with-custom-fields
```

**Pause**: Let viewers see the seeding summary with emoji indicators

**Script**: "We've loaded 50 issues with custom fields like story points and estimated hours."

---

### PART 3: Inspecting Data (45 seconds)

#### Step 4: View Storage Statistics
**Script**: "Now let's see what data we have."

```bash
./bin/set.exe inspect
```

**Pause**: Show the rounded box with statistics

**Script**: "Notice the clean, modern interface with rounded corners. We have 50 issues, and 19 include custom fields from GitHub Projects."

---

#### Step 5: List Issues
**Script**: "We can list all issues in a table."

```bash
./bin/set.exe inspect --list --limit 10
```

**Pause**: Show the table

**Script**: "Here's a quick view of our historical tasks."

---

### PART 4: Single Task Estimation (90 seconds)

#### Step 6: Simple Estimation
**Script**: "Now the main feature - AI-powered estimation. Let's estimate a task."

```bash
./bin/set.exe estimate "Add user authentication"
```

**Pause**: Let the AI process (6-10 seconds)

**Script**: "SET uses OpenAI's GPT model combined with our historical data. Look at this beautiful output!"

**Point out**:
- "The rounded box shows our estimate: 15 hours, Large size, 8 story points"
- "60% confidence with a visual progress bar"
- "Detailed analysis explaining the reasoning"
- "Key assumptions the AI is making"
- "Potential risks highlighted with lightning bolts"
- "Actionable recommendations"

---

#### Step 7: Estimation with Context
**Script**: "Let's try a more detailed task with description and labels."

```bash
./bin/set.exe estimate "Implement dark mode toggle" --description "Add theme switcher with localStorage persistence" --labels frontend,ui
```

**Pause**: Show the task box and estimation

**Script**: "Notice how labels and description give us more context, leading to a more accurate estimate of 6 hours."

---

#### Step 8: Show Similar Tasks
**Script**: "We can also see similar historical tasks that influenced the estimate."

```bash
./bin/set.exe estimate "Fix memory leak" --description "Cache system not releasing resources" --labels bug,performance --show-similar
```

**Pause**: Show the similar tasks table

**Script**: "SET found similar bugs in our historical data and used them for comparison."

---

### PART 5: Batch Processing (60 seconds)

#### Step 9: Show Batch File
**Script**: "For sprint planning, we can estimate multiple tasks at once. I've prepared a sprint backlog with 5 real-world tasks."

**Action**: Quickly show the file content
```bash
type examples\batch_example.json
```

**Script**: "We have OAuth authentication, a memory leak bug fix, PDF export feature, database refactoring, and integration tests."

---

#### Step 10: Run Batch Estimation (JSON)
**Script**: "Let's estimate all five tasks in parallel using the JSON format."

```bash
./bin/set.exe batch --file examples\batch_example.json
```

**Pause**: Let it process all 5 tasks (25-30 seconds)

**Script**: "Watch as SET processes all tasks simultaneously using 5 workers."

**Pause**: Show the results

**Point out**:
- "Overall statistics in a rounded box - total estimated hours for Sprint 15"
- "Size distribution across XS, S, M, L, XL"
- "Confidence distribution showing reliability"
- "Detailed results table with all 5 tasks"

**Script**: "This is perfect for sprint planning - estimate your entire backlog in seconds!"

---

#### Step 10b: Try CSV Format (Optional)
**Script**: "We also support CSV format for easy spreadsheet integration."

```bash
./bin/set.exe batch --file examples\batch_example.csv
```

**Script**: "Same tasks, different format - SET handles both!"

---

### PART 6: Export & Integration (45 seconds)

#### Step 11: Export to CSV
**Script**: "Let's export our data for analysis in Excel or other tools."

```bash
./bin/set.exe export --format csv --output demo-export.csv
```

**Script**: "We've exported all 50 issues to CSV."

**Action**: Quickly open the CSV file
```bash
type demo-export.csv | findstr /n "^" | findstr "^[1-5]:"
```

**Script**: "You can import this into JIRA, Excel, or any project management tool."

---

#### Step 12: Export Batch Results
**Script**: "We can also export batch results in multiple formats for documentation."

```bash
./bin/set.exe batch --file examples\batch_example.json --output sprint-15-estimate.md --format markdown
```

**Script**: "Now we have a markdown report ready to share with the team or add to documentation."

**Action**: Quickly show the markdown file
```bash
type sprint-15-estimate.md | findstr /n "^" | findstr "^[1-15]:"
```

---

### PART 7: Advanced Features (30 seconds)

#### Step 13: Configuration
**Script**: "SET is highly configurable. Let's check our settings."

```bash
./bin/set.exe configure list
```

**Script**: "You can configure AI providers, API keys, repositories, and estimation parameters."

---

### PART 8: UI Showcase (30 seconds)

**Script**: "Let me highlight the Modern Minimal UI we've designed."

**Action**: Run one more estimate to show the UI
```bash
./bin/set.exe estimate "Refactor authentication module" --labels refactoring,backend
```

**Point out while it's processing**:
- "Rounded box corners for a modern look"
- "Section markers with perfect vertical alignment"
- "Visual progress bars using ASCII blocks"
- "Strategic emoji use - only where it adds meaning"
- "Clean, professional aesthetic that doesn't look AI-generated"

---

### CLOSING (20 seconds)

**Script**:
> "That's SET CLI! An AI-powered software estimation tool with:
> - Accurate estimates using GPT models and historical data
> - Beautiful, modern terminal UI
> - Batch processing for sprint planning
> - Multiple export formats
> - Full GitHub integration
>
> Perfect for developers, scrum masters, and product owners. Thanks for watching!"

**Action**: Clear screen and show final command
```bash
./bin/set.exe --help
```

---

## Quick Reference - Copy-Paste Commands

### Setup
```bash
cd "C:\Users\Inácio Moraes\Documents\GitHub\plf-es-2025-2-tcci-0393100-dev-set\Codigo"
cls
```

### Demo Commands (in order)
```bash
# 1. Help
./bin/set.exe --help

# 2. Version
./bin/set.exe version

# 3. Seed data
./bin/set.exe dev seed --count 50 --with-custom-fields

# 4. Inspect
./bin/set.exe inspect

# 5. List issues
./bin/set.exe inspect --list --limit 10

# 6. Simple estimate
./bin/set.exe estimate "Add user authentication"

# 7. Detailed estimate
./bin/set.exe estimate "Implement dark mode toggle" --description "Add theme switcher with localStorage persistence" --labels frontend,ui

# 8. Estimate with similar tasks
./bin/set.exe estimate "Fix memory leak" --description "Cache system not releasing resources" --labels bug,performance --show-similar

# 9. Show batch file
type examples\batch_example.json

# 10. Batch estimation (JSON)
./bin/set.exe batch --file examples\batch_example.json

# 10b. Batch estimation (CSV) - Optional
./bin/set.exe batch --file examples\batch_example.csv

# 11. Export to CSV
./bin/set.exe export --format csv --output demo-export.csv

# 12. Show CSV (first 5 lines)
type demo-export.csv | findstr /n "^" | findstr "^[1-5]:"

# 13. Batch export to Markdown
./bin/set.exe batch --file examples\batch_example.json --output sprint-15-estimate.md --format markdown

# 13b. Show markdown output
type sprint-15-estimate.md | findstr /n "^" | findstr "^[1-15]:"

# 14. Configuration
./bin/set.exe configure list

# 15. Final showcase estimate
./bin/set.exe estimate "Refactor authentication module" --labels refactoring,backend
```

---

## Pro Tips for Recording

### Before Recording
1. ✅ Close all unnecessary applications
2. ✅ Set terminal to 80x40 or larger
3. ✅ Use a clean terminal theme (dark background recommended)
4. ✅ Test all commands beforehand to ensure they work
5. ✅ Have test-tasks.json ready in the current directory
6. ✅ Verify OpenAI API key is configured

### During Recording
1. **Speak clearly and slowly** - viewers need to follow along
2. **Pause after each output** - let viewers read the results
3. **Use mouse/cursor to highlight** - point to interesting sections
4. **Don't rush AI calls** - the processing time shows it's real
5. **Show enthusiasm** - this is a cool tool!

### Recording Settings
- **Resolution**: 1920x1080 minimum
- **FPS**: 30fps or 60fps
- **Audio**: Clear voice, minimal background noise
- **Highlight**: Use a cursor highlighter for emphasis

### Common Issues
- **API Rate Limits**: If using free OpenAI tier, wait 60s between estimates
- **Slow Responses**: AI calls take 5-10 seconds - this is normal
- **Terminal Size**: Make sure all box borders fit on screen (60 chars wide)

---

## Alternative: Quick Demo (2 minutes)

If you need a shorter version:

```bash
# Setup
cd "C:\Users\Inácio Moraes\Documents\GitHub\plf-es-2025-2-tcci-0393100-dev-set\Codigo"
cls

# Show help
./bin/set.exe --help

# Load sample data
./bin/set.exe dev seed --count 50 --with-custom-fields

# Single estimate
./bin/set.exe estimate "Add user authentication" --description "OAuth 2.0 integration" --labels backend,security

# Batch estimate with real sprint data
./bin/set.exe batch --file examples\batch_example.json

# Export results
./bin/set.exe batch --file examples\batch_example.json --output sprint-estimate.csv --format csv

# Done!
```

**Script**:
> "SET CLI: AI-powered software estimation in your terminal. Load historical data, estimate tasks with GPT-4, batch process your backlog. Clean UI, accurate estimates, perfect for agile teams. That's it!"

---

## Customization Ideas

### Add Your Own Flavor
- Show a real GitHub repository sync instead of sample data
- Use tasks from your actual project
- Compare estimates with actual time spent
- Show how estimates improve with more historical data

### Advanced Demo
```bash
# Sync from real GitHub repo (if configured)
./bin/set.exe sync --custom-fields

# Inspect specific issue
./bin/set.exe inspect --issue 1234

# Export to JIRA format
./bin/set.exe export --format jira --output jira-import.csv

# Different AI models (if configured)
./bin/set.exe configure --ai-model gpt-4-turbo
```

---

## Post-Production Checklist

- [ ] Add title card: "SET CLI - Software Estimation Tool"
- [ ] Add captions for key features
- [ ] Highlight important UI elements (rounded boxes, progress bars)
- [ ] Add background music (optional, keep it subtle)
- [ ] Include GitHub link in description
- [ ] Add timestamps in video description
- [ ] Consider adding zoom-ins for detailed sections

---

## Video Timestamps (Suggested)

```
0:00 - Introduction
0:15 - Getting Started (Help & Version)
1:00 - Loading Sample Data
1:30 - Inspecting Data
2:15 - Single Task Estimation
3:45 - Estimation with Context
4:30 - Batch Processing
5:30 - Export & Integration
6:15 - UI Showcase
6:45 - Closing
```

---

Good luck with your demo! 🎬🚀

**Remember**: The Modern Minimal UI is your star feature - make sure to highlight those beautiful rounded boxes and perfect alignment! ✨
