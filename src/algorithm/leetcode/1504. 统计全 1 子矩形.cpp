class Solution
{
public:
    int numSubmat(vector<vector<int>> &mat)
    {
        int m = mat.size(), n = mat[0].size();
        int ans = 0;
        for (int top = 0; top < m; ++top)
        {
            vector<int> a(n);
            for (int bottom = top; bottom < m; ++bottom)
            {
                int h = bottom - top + 1;
                int last = -1;
                for (int i = 0; i < n; ++i)
                {
                    a[i] += mat[bottom][i];
                    if (a[i] != h)
                    {
                        last = i;
                    }
                    else
                    {
                        ans += i - last;
                    }
                }
            }
        }
        return ans;
    }
};