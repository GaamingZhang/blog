class Solution
{
public:
    string removeDuplicateLetters(string s)
    {
        unordered_map<char, int> count, used, removed;
        for (int i = 0; i < s.size(); ++i)
            ++count[s[i]];
        vector<char> st;
        for (int i = 0; i < s.size(); ++i)
        {
            if (used[s[i]] > 0)
            {
                ++removed[s[i]];
                continue;
            }
            while (!st.empty() && st.back() > s[i] && count[st.back()] - used[st.back()] - removed[st.back()] > 0)
            {
                --used[st.back()];
                ++removed[st.back()];
                st.pop_back();
            }
            ++used[s[i]];
            st.push_back(s[i]);
        }
        string result;
        for (int i = 0; i < st.size(); ++i)
            result += st[i];
        return result;
    }
};