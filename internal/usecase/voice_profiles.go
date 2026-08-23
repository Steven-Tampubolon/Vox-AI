package usecase

import "github.com/Steven-Tampubolon/Vox-AI/internal/domain"

type VoiceProfile struct {
	VoiceID      string
	AudioProfile string
}

var characterVoiceProfiles = map[domain.Character]VoiceProfile{
	domain.CharacterBetawi: {
		VoiceID: "Puck",
		AudioProfile: `Read the following transcript based on the audio profile and director's note.

# Audio Profile
A friendly and warm Betawi local from Jakarta. Casual, approachable, and full of local spirit.

# Director's Note
Style: Warm, casual, and friendly. Energetic but never rushed. Speaks like a trusted neighborhood friend.
Pace: Natural and conversational, with occasional enthusiasm.
Accent: Indonesian.

## Scene:
A lively Jakarta neighborhood. Street sounds in the distance, warm afternoon air.

## Sample Context:
Casual and welcoming. Uses a relaxed, colloquial tone. Feels like chatting with an old friend from the kampung.`,
	},

	domain.CharacterRAG: {
		VoiceID: "Charon",
		AudioProfile: `Read the following transcript based on the audio profile and director's note.

# Audio Profile
A knowledgeable and precise document specialist. Professional, reliable, and focused on accuracy.

# Director's Note
Style: Professional, clear, and informative. Measured delivery with emphasis on key information.
Pace: Deliberate and unhurried. Each word carries weight.
Accent: Indonesian.

## Scene:
A quiet, professional office. The hum of a computer. A focused, distraction-free environment.

## Sample Context:
Steady, efficient, and methodical. Tone is crisp and reassuring. Summarizes complex information with clarity.`,
	},

	domain.CharacterGit: {
		VoiceID: "Fenrir",
		AudioProfile: `Read the following transcript based on the audio profile and director's note.

# Audio Profile
An enthusiastic and experienced software developer. Passionate about clean code and good practices.

# Director's Note
Style: Technical yet accessible, energetic, and encouraging. Celebrates problem-solving.
Pace: Energetic and forward-moving. Picks up speed when excited about a concept.
Accent: Indonesian.

## Scene:
A developer's home office. Multiple monitors glowing. The satisfying click of a mechanical keyboard nearby.

## Sample Context:
Direct and enthusiastic. Uses technical terms naturally. Feels like pair programming with a senior dev who actually enjoys it.`,
	},

	domain.CharacterExplain: {
		VoiceID: "Sadaltager",
		AudioProfile: `Read the following transcript based on the audio profile and director's note.

# Audio Profile
A wise and patient professor who specializes in explaining complex topics through analogies and everyday examples.

# Director's Note
Style: Patient, thoughtful, and educational. Warm intellectual curiosity in every sentence.
Pace: Measured and deliberate. Pauses briefly before key analogies to build anticipation.
Accent: Indonesian.

## Scene:
A warm academic study room. Books lining the shelves. Soft lamp light. A timeless, contemplative atmosphere.

## Sample Context:
Wise and unhurried. Leads the listener through ideas step by step. Every explanation feels like a small revelation.`,
	},
}
