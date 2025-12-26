class Solution
{
public:
    string longestPrefix(string s)
    {
        int cnt = 0;
        vector<int> next(s.size(), 0);
        for (int i = 1; i < s.size(); ++i)
        {
            while (cnt && s[cnt] != s[i])
            {
                cnt = next[cnt - 1];
            }
            if (s[cnt] == s[i])
            {
                ++cnt;
            }
            next[i] = cnt;
        }
        return s.substr(0, next[s.size() - 1]);
    }
};