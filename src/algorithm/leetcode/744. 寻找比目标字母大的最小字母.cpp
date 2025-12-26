class Solution
{
public:
    char nextGreatestLetter(vector<char> &letters, char target)
    {
        int index = search(letters, target + 1);
        if (index == letters.size())
            return letters[0];
        return letters[index];
    }
    int search(vector<char> &letters, char target)
    {
        int left = 0, right = letters.size();
        while (left < right)
        {
            int mid = left + (right - left) / 2;
            if (letters[mid] >= target)
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