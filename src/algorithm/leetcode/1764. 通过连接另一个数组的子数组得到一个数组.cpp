class Solution
{
public:
    bool canChoose(vector<vector<int>> &groups, vector<int> &nums)
    {
        int numsIndex = 0;
        for (int i = 0; i < groups.size(); ++i)
        {
            vector<int> next = getNext(groups[i]);
            int cnt = 0;
            while (numsIndex < nums.size())
            {
                while (cnt && nums[numsIndex] != groups[i][cnt])
                {
                    cnt = next[cnt - 1];
                }
                if (nums[numsIndex] == groups[i][cnt])
                {
                    ++cnt;
                }
                ++numsIndex;
                if (cnt == groups[i].size())
                {
                    if (i == groups.size() - 1)
                        return true;
                    break;
                }
            }
        }
        return false;
    }

private:
    vector<int> getNext(vector<int> &nums)
    {
        int cnt = 0;
        vector<int> next(nums.size(), 0);
        for (int i = 1; i < nums.size(); ++i)
        {
            while (cnt && nums[cnt] != nums[i])
            {
                cnt = next[cnt - 1];
            }
            if (nums[cnt] == nums[i])
            {
                ++cnt;
            }
            next[i] = cnt;
        }
        return next;
    }
};