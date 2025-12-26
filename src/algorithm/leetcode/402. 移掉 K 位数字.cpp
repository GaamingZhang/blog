class Solution
{
public:
    string removeKdigits(string num, int k)
    {
        vector<char> st;
        for (int i = 0; i < num.size(); ++i)
        {
            while (!st.empty() && st.back() > num[i] && k)
            {
                st.pop_back();
                --k;
            }
            st.push_back(num[i]);
        }
        while (k > 0)
        {
            st.pop_back();
            --k;
        }
        string ans = "";
        bool isLeadingZero = true;
        for (int i = 0; i < st.size(); ++i)
        {
            if (isLeadingZero && st[i] == '0')
            {
                continue;
            }
            isLeadingZero = false;
            ans += st[i];
        }
        return ans == "" ? "0" : ans;
    }
};