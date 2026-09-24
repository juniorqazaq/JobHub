import type {
  CandidateProfile,
  CandidateProfileInput,
} from "../api/models/candidate";

export function editableProfile(
  profile: CandidateProfile,
): CandidateProfileInput {
  const { userId, resume, completion, ...input } = profile;
  void userId;
  void resume;
  void completion;
  return input;
}
