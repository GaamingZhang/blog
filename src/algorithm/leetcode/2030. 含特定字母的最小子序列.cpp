class Solution
{
public:
    string smallestSubsequence(string s, int k, char letter, int repetition)
    {
        int letterCount = 0;
        for (int i = 0; i < s.size(); ++i)
        {
            if (s[i] == letter)
                ++letterCount;
        }
        vector<char> st;
        int removeKNum = max(0, (int)(s.size() - k)), removeLetterNum = max(0, letterCount - repetition);
        for (int i = 0; i < s.size(); ++i)
        {
            while (!st.empty() && st.back() > s[i] && removeKNum)
            {
                if (st.back() == letter && removeLetterNum <= 0)
                    break;
                if (st.back() == letter)
                    --removeLetterNum;
                --removeKNum;
                st.pop_back();
            }
            st.push_back(s[i]);
        }
        string res;
        letterCount = 0;
        for (int i = 0; i < st.size() && res.size() < k; ++i)
        {
            if (st[i] == letter && letterCount < repetition)
            {
                res += st[i];
                ++letterCount;
            }
            else if (res.size() + repetition - letterCount < k)
                res += st[i];
        }
        return res;
    }
};