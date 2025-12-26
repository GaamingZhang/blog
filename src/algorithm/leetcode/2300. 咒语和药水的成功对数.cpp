class Solution
{
public:
    vector<int> successfulPairs(vector<int> &spells, vector<int> &potions, long long success)
    {
        vector<int> pairs(spells.size(), 0);
        sort(potions.begin(), potions.end(), less<int>());
        for (int i = 0; i < spells.size(); ++i)
        {
            long long target = 0;
            if (spells[i] == 1)
            {
                target = success;
            }
            else
            {
                target = success % spells[i] == 0 ? success / spells[i] : success / spells[i] + 1;
            }
            pairs[i] = potions.size() - lowerBound(potions, target);
        }
        return pairs;
    }
    int lowerBound(vector<int> &nums, long long target)
    {
        int left = 0, right = nums.size();
        while (left < right)
        {
            int mid = left + (right - left) / 2;
            if (nums[mid] >= target)
            {
                right = mid;
            }
            else
            {
                left = mid + 1;
            }
        }
        return left;
    }
};