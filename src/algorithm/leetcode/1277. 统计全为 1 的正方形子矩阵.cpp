class Solution
{
public:
    int countSquares(vector<vector<int>> &matrix)
    {
        int m = matrix.size(), n = matrix[0].size();
        int ans = 0;
        for (int top = 0; top < m; ++top)
        {
            vector<int> tmp(n);
            for (int bottom = top; bottom < m; ++bottom)
            {
                int h = bottom - top + 1;
                int last = -1;
                for (int i = 0; i < n; ++i)
                {
                    tmp[i] += matrix[bottom][i];
                    if (tmp[i] != h)
                    {
                        last = i;
                    }
                    else if (i - last >= h)
                    {
                        ++ans;
                    }
                }
            }
        }
        return ans;
    }
};