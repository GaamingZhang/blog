class Solution
{
public:
    int strStr(string haystack, string needle)
    {
        int m = needle.size(), cnt = 0;
        vector<int> pi(m, 0);
        for (int i = 1; i < m; ++i)
        {
            char tmp = needle[i];
            while (cnt && needle[cnt] != tmp)
            {
                cnt = pi[cnt - 1];
            }
            if (needle[cnt] == tmp)
            {
                ++cnt;
            }
            pi[i] = cnt;
        }

        cnt = 0;
        for (int i = 0; i < haystack.size(); ++i)
        {
            char tmp = haystack[i];
            while (cnt && needle[cnt] != tmp)
            {
                cnt = pi[cnt - 1];
            }
            if (needle[cnt] == tmp)
            {
                ++cnt;
            }
            if (cnt == m)
            {
                return i - m + 1;
            }
        }
        return -1;
    }
};