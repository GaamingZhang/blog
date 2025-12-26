class Solution
{
public:
    bool rotateString(string s, string goal)
    {
        if (s.size() != goal.size())
            return false;
        vector<int> next = getNext(goal);
        string ss = s + s;
        int cnt = 0;
        for (int i = 0; i < ss.size(); ++i)
        {
            char tmp = ss[i];
            while (cnt && goal[cnt] != tmp)
            {
                cnt = next[cnt - 1];
            }
            if (goal[cnt] == tmp)
            {
                ++cnt;
            }
            if (cnt == goal.size())
            {
                return true;
            }
        }
        return false;
    }

private:
    vector<int> getNext(const string &goal)
    {
        int cnt = 0;
        vector<int> next(goal.size(), 0);
        for (int i = 1; i < goal.size(); ++i)
        {
            char tmp = goal[i];
            while (cnt && goal[cnt] != tmp)
            {
                cnt = next[cnt - 1];
            }
            if (goal[cnt] == tmp)
            {
                ++cnt;
            }
            next[i] = cnt;
        }
        return next;
    }
};