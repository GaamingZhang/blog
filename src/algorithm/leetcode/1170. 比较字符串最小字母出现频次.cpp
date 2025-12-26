class Solution
{
public:
    vector<int> numSmallerByFrequency(vector<string> &queries, vector<string> &words)
    {
        vector<int> result(queries.size(), 0), wordsFFunc(words.size(), 0);
        for (int i = 0; i < words.size(); ++i)
        {
            wordsFFunc[i] = ffunc(words[i]);
        }
        sort(wordsFFunc.begin(), wordsFFunc.end(), greater<int>());
        for (int i = 0; i < queries.size(); ++i)
        {
            int queriesFFunc = ffunc(queries[i]);
            result[i] = lower_bound(wordsFFunc.begin(), wordsFFunc.end(), queriesFFunc, greater<int>()) - wordsFFunc.begin();
        }
        return result;
    }
    int ffunc(string word)
    {
        char minChar = 'z';
        int result = 0;
        for (int i = 0; i < word.size(); ++i)
        {
            if (word[i] < minChar)
            {
                minChar = word[i];
                result = 1;
            }
            else if (word[i] == minChar)
            {
                ++result;
            }
        }
        return result;
    }
};