class Solution
{
public:
    string smallestSubsequence(string s)
    {
        vector<int> count(26, 0), visited(26, 0), removed(26, 0);
        for (int i = 0; i < s.size(); ++i)
        {
            ++count[s[i] - 'a'];
        }
        vector<char> st;
        string res;
        for (int i = 0; i < s.size(); ++i)
        {
            if (visited[s[i] - 'a'] > 0)
            {
                ++removed[s[i] - 'a'];
                continue;
            }
            while (!st.empty() && st.back() > s[i] && count[st.back() - 'a'] - removed[st.back() - 'a'] - visited[st.back() - 'a'] > 0)
            {
                ++removed[st.back() - 'a'];
                --visited[st.back() - 'a'];
                st.pop_back();
            }
            st.push_back(s[i]);
            ++visited[s[i] - 'a'];
        }
        for (int i = 0; i < st.size(); ++i)
        {
            res += st[i];
        }
        return res;
    }
};